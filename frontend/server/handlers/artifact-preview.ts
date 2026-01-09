// Copyright 2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import fetch from 'node-fetch';
import { Handler, Request, Response } from 'express';
import { UIConfigs } from '../configs';
import { getAddress, PreviewStream, parseJSONString } from '../utils';
import { getConfigMap, getK8sSecret } from '../k8s-helper';
import { createMinioClient, getObjectStream } from '../minio-helper';
import { Storage, StorageOptions } from '@google-cloud/storage';
import { CredentialBody } from 'google-auth-library/build/src/auth/credentials';
import * as yaml from 'js-yaml';

// ============================================================================
// Types for bucket provider configurations (mirrors backend/src/v2/config/)
// ============================================================================

const CONFIG_MAP_NAME = 'kfp-launcher';
const PROVIDERS_KEY = 'providers';

export interface SessionInfo {
  provider: 'minio' | 's3' | 'gs';
  params: Record<string, string>;
}

export interface BucketProviders {
  minio?: S3ProviderConfig;
  s3?: S3ProviderConfig;
  gcs?: GCSProviderConfig;
}

export interface S3ProviderConfig {
  default?: S3ProviderDefault;
  overrides?: S3Override[];
}

export interface S3ProviderDefault {
  endpoint?: string;
  credentials?: S3Credentials;
  region?: string;
  disableSSL?: boolean;
  forcePathStyle?: boolean;
  maxRetries?: number;
}

export interface S3Credentials {
  fromEnv?: boolean;
  secretRef?: S3SecretRef;
}

export interface S3SecretRef {
  secretName: string;
  accessKeyKey: string;
  secretKeyKey: string;
}

export interface S3Override extends S3ProviderDefault {
  bucketName: string;
  keyPrefix?: string;
}

export interface GCSProviderConfig {
  default?: GCSProviderDefault;
  overrides?: GCSOverride[];
}

export interface GCSProviderDefault {
  credentials?: GCSCredentials;
}

export interface GCSCredentials {
  fromEnv?: boolean;
  secretRef?: GCSSecretRef;
}

export interface GCSSecretRef {
  secretName: string;
  tokenKey: string;
}

export interface GCSOverride extends GCSProviderDefault {
  bucketName: string;
  keyPrefix?: string;
}

// Artifact response from backend API
export interface BackendArtifact {
  artifact_id?: string;
  name?: string;
  uri?: string;
  namespace?: string;
  type?: string;
}

// Storage path parsed from URI
export interface StoragePath {
  source: 'minio' | 's3' | 'gcs' | 'http' | 'https' | 'volume';
  bucket: string;
  key: string;
}

// ============================================================================
// Launcher ConfigMap Cache
// ============================================================================

interface ConfigCache {
  data: BucketProviders | null;
  timestamp: number;
}

const configCacheByNamespace: Map<string, ConfigCache> = new Map();
const CACHE_TTL_MS = 60000; // 1 minute

/**
 * Fetches the kfp-launcher ConfigMap and parses the providers section.
 */
async function fetchLauncherConfigMap(namespace: string): Promise<BucketProviders | null> {
  // Check cache
  const cached = configCacheByNamespace.get(namespace);
  if (cached && Date.now() - cached.timestamp < CACHE_TTL_MS) {
    return cached.data;
  }

  const [configMap, error] = await getConfigMap(CONFIG_MAP_NAME, namespace);
  if (error || !configMap?.data?.[PROVIDERS_KEY]) {
    console.debug(
      `kfp-launcher configmap not found or missing providers key in namespace ${namespace}`,
    );
    configCacheByNamespace.set(namespace, { data: null, timestamp: Date.now() });
    return null;
  }

  try {
    const providers = yaml.load(configMap.data[PROVIDERS_KEY]) as BucketProviders;
    configCacheByNamespace.set(namespace, { data: providers, timestamp: Date.now() });
    return providers;
  } catch (err) {
    console.error('Failed to parse bucket providers from kfp-launcher configmap:', err);
    configCacheByNamespace.set(namespace, { data: null, timestamp: Date.now() });
    return null;
  }
}

/**
 * Parses the storage provider from a URI.
 */
function parseProviderFromPath(uri: string): string | null {
  const match = uri.match(/^([a-z0-9]+):\/\//);
  return match ? match[1] : null;
}

/**
 * Parses bucket name and key from a URI.
 */
function parseBucketAndKey(uri: string): { bucket: string; key: string } | null {
  const match = uri.match(/^[a-z0-9]+:\/\/([^/?]+)(?:\/([^?]*))?/);
  if (!match) return null;
  return { bucket: match[1], key: match[2] || '' };
}

/**
 * Gets session info for a storage URI, similar to backend's GetStoreSessionInfo.
 */
async function getStoreSessionInfo(
  uri: string,
  namespace: string,
): Promise<SessionInfo | null> {
  const provider = parseProviderFromPath(uri);
  if (!provider) {
    return null;
  }

  const bucketProviders = await fetchLauncherConfigMap(namespace);
  const bucketInfo = parseBucketAndKey(uri);

  switch (provider) {
    case 'minio':
      return getMinioSessionInfo(bucketInfo, bucketProviders?.minio);
    case 's3':
      return getS3SessionInfo(bucketInfo, bucketProviders?.s3);
    case 'gs':
    case 'gcs':
      return getGCSSessionInfo(bucketInfo, bucketProviders?.gcs);
    default:
      return null;
  }
}

function getS3SessionInfo(
  bucketInfo: { bucket: string; key: string } | null,
  config: S3ProviderConfig | undefined,
): SessionInfo {
  const params: Record<string, string> = { fromEnv: 'true' };

  if (!config) {
    return { provider: 's3', params };
  }

  // Check for matching override
  if (bucketInfo && config.overrides) {
    for (const override of config.overrides) {
      if (
        override.bucketName === bucketInfo.bucket &&
        (!override.keyPrefix || bucketInfo.key.startsWith(override.keyPrefix))
      ) {
        return buildS3SessionParams('s3', override);
      }
    }
  }

  // Use default config
  if (config.default) {
    return buildS3SessionParams('s3', config.default);
  }

  return { provider: 's3', params };
}

function getMinioSessionInfo(
  bucketInfo: { bucket: string; key: string } | null,
  config: S3ProviderConfig | undefined,
): SessionInfo {
  const params: Record<string, string> = { fromEnv: 'true' };

  if (!config) {
    // Default Minio config
    return { provider: 'minio', params };
  }

  // Check for matching override
  if (bucketInfo && config.overrides) {
    for (const override of config.overrides) {
      if (
        override.bucketName === bucketInfo.bucket &&
        (!override.keyPrefix || bucketInfo.key.startsWith(override.keyPrefix))
      ) {
        return buildS3SessionParams('minio', override);
      }
    }
  }

  // Use default config
  if (config.default) {
    return buildS3SessionParams('minio', config.default);
  }

  return { provider: 'minio', params };
}

function buildS3SessionParams(
  provider: 'minio' | 's3',
  config: S3ProviderDefault,
): SessionInfo {
  const params: Record<string, string> = {};

  if (config.credentials?.fromEnv) {
    params.fromEnv = 'true';
  } else if (config.credentials?.secretRef) {
    params.fromEnv = 'false';
    params.secretName = config.credentials.secretRef.secretName;
    params.accessKeyKey = config.credentials.secretRef.accessKeyKey;
    params.secretKeyKey = config.credentials.secretRef.secretKeyKey;
  } else {
    params.fromEnv = 'true';
  }

  if (config.endpoint) params.endpoint = config.endpoint;
  if (config.region) params.region = config.region;
  if (config.disableSSL !== undefined) params.disableSSL = String(config.disableSSL);
  if (config.forcePathStyle !== undefined) params.forcePathStyle = String(config.forcePathStyle);
  if (config.maxRetries !== undefined) params.maxRetries = String(config.maxRetries);

  return { provider, params };
}

function getGCSSessionInfo(
  bucketInfo: { bucket: string; key: string } | null,
  config: GCSProviderConfig | undefined,
): SessionInfo {
  const params: Record<string, string> = { fromEnv: 'true' };

  if (!config) {
    return { provider: 'gs', params };
  }

  // Check for matching override
  if (bucketInfo && config.overrides) {
    for (const override of config.overrides) {
      if (
        override.bucketName === bucketInfo.bucket &&
        (!override.keyPrefix || bucketInfo.key.startsWith(override.keyPrefix))
      ) {
        return buildGCSSessionParams(override);
      }
    }
  }

  // Use default config
  if (config.default) {
    return buildGCSSessionParams(config.default);
  }

  return { provider: 'gs', params };
}

function buildGCSSessionParams(config: GCSProviderDefault): SessionInfo {
  const params: Record<string, string> = {};

  if (config.credentials?.fromEnv) {
    params.fromEnv = 'true';
  } else if (config.credentials?.secretRef) {
    params.fromEnv = 'false';
    params.secretName = config.credentials.secretRef.secretName;
    params.tokenKey = config.credentials.secretRef.tokenKey;
  } else {
    params.fromEnv = 'true';
  }

  return { provider: 'gs', params };
}

// ============================================================================
// Storage Path Parsing
// ============================================================================

/**
 * Parses a storage URI into its components.
 */
function parseStoragePath(uri: string): StoragePath | null {
  const patterns: [RegExp, StoragePath['source']][] = [
    [/^gs:\/\/([^/]+)\/(.+)$/, 'gcs'],
    [/^gcs:\/\/([^/]+)\/(.+)$/, 'gcs'],
    [/^minio:\/\/([^/]+)\/(.+)$/, 'minio'],
    [/^s3:\/\/([^/]+)\/(.+)$/, 's3'],
    [/^http:\/\/([^/]+)\/(.+)$/, 'http'],
    [/^https:\/\/([^/]+)\/(.+)$/, 'https'],
    [/^volume:\/\/([^/]+)\/(.+)$/, 'volume'],
  ];

  for (const [pattern, source] of patterns) {
    const match = uri.match(pattern);
    if (match) {
      return { source, bucket: match[1], key: match[2] };
    }
  }

  return null;
}

// ============================================================================
// Artifact Preview Handler
// ============================================================================

interface ArtifactPreviewHandlerOptions {
  options: UIConfigs;
}

/**
 * Fetches artifact metadata from the backend API with auth header forwarding.
 * The backend performs RBAC authorization check.
 */
async function fetchArtifactFromBackend(
  artifactId: string,
  req: Request,
  apiServerAddress: string,
  authConfig: { kubeflowUserIdHeader: string },
): Promise<BackendArtifact> {
  const headers: Record<string, string> = {};

  // Forward authentication headers
  const userIdHeader = authConfig.kubeflowUserIdHeader.toLowerCase();
  if (req.headers[userIdHeader]) {
    headers[authConfig.kubeflowUserIdHeader] = req.headers[userIdHeader] as string;
  }

  // Forward Bearer token if present
  if (req.headers.authorization) {
    headers['Authorization'] = req.headers.authorization as string;
  }

  const url = `${apiServerAddress}/apis/v2beta1/artifacts/${encodeURIComponent(artifactId)}`;
  console.log(`Fetching artifact metadata from backend: ${url}`);

  const response = await fetch(url, {
    headers,
  });

  if (!response.ok) {
    const errorText = await response.text();
    const error = new Error(errorText) as Error & { status: number };
    error.status = response.status;
    throw error;
  }

  return (await response.json()) as BackendArtifact;
}

/**
 * Handler for GET /artifacts/:artifactId/preview
 *
 * 1. Forwards auth headers to backend API to fetch artifact metadata
 * 2. Backend performs RBAC check and returns artifact with namespace/uri
 * 3. Retrieves session info from kfp-launcher ConfigMap
 * 4. Fetches artifact content from storage backend
 */
export function getArtifactPreviewHandler({ options }: ArtifactPreviewHandlerOptions): Handler {
  const apiServerAddress = getAddress(options.pipeline);
  const serverNamespace = options.server.serverNamespace;

  return async (req: Request, res: Response) => {
    const { artifactId } = req.params;
    const peek = parseInt(req.query.peek as string, 10) || 256;

    if (!artifactId) {
      res.status(400).json({ error: 'Artifact ID is required' });
      return;
    }

    try {
      // Step 1: Call backend API to get artifact metadata (with auth headers)
      const artifact = await fetchArtifactFromBackend(
        artifactId,
        req,
        apiServerAddress,
        options.auth,
      );

      if (!artifact) {
        res.status(404).json({ error: 'Artifact not found' });
        return;
      }

      if (!artifact.uri) {
        res.status(400).json({ error: 'Artifact has no URI' });
        return;
      }

      // Step 2: Parse storage path from artifact URI
      const storagePath = parseStoragePath(artifact.uri);
      if (!storagePath) {
        res.status(400).json({ error: `Invalid artifact URI: ${artifact.uri}` });
        return;
      }

      // Step 3: Get session info from kfp-launcher ConfigMap (using artifact's namespace)
      const namespace = artifact.namespace || serverNamespace;
      const sessionInfo = await getStoreSessionInfo(artifact.uri, namespace);

      // Step 4: Fetch artifact content based on storage type
      await fetchAndStreamArtifact(
        req,
        res,
        storagePath,
        sessionInfo,
        namespace,
        peek,
        options,
        false, // tryExtract for preview
      );
    } catch (error: any) {
      console.error('Error fetching artifact preview:', error);

      if (error.status === 401) {
        res.status(401).json({
          error: 'Authentication required',
          details: 'You must be logged in to access this artifact',
        });
        return;
      }

      if (error.status === 403) {
        res.status(403).json({
          error: 'Access denied',
          details: 'You do not have permission to access this artifact',
        });
        return;
      }

      if (error.status === 404) {
        res.status(404).json({
          error: 'Artifact not found',
          details: error.message,
        });
        return;
      }

      res.status(500).json({
        error: 'Failed to fetch artifact preview',
        details: error.message,
      });
    }
  };
}

/**
 * Handler for GET /artifacts/:artifactId/download
 * Downloads the full artifact content.
 */
export function getArtifactDownloadHandler({ options }: ArtifactPreviewHandlerOptions): Handler {
  const apiServerAddress = getAddress(options.pipeline);
  const serverNamespace = options.server.serverNamespace;

  return async (req: Request, res: Response) => {
    const { artifactId } = req.params;

    if (!artifactId) {
      res.status(400).json({ error: 'Artifact ID is required' });
      return;
    }

    try {
      const artifact = await fetchArtifactFromBackend(
        artifactId,
        req,
        apiServerAddress,
        options.auth,
      );

      if (!artifact || !artifact.uri) {
        res.status(404).json({ error: 'Artifact not found or has no URI' });
        return;
      }

      const storagePath = parseStoragePath(artifact.uri);
      if (!storagePath) {
        res.status(400).json({ error: `Invalid artifact URI: ${artifact.uri}` });
        return;
      }

      const namespace = artifact.namespace || serverNamespace;
      const sessionInfo = await getStoreSessionInfo(artifact.uri, namespace);

      // Set download headers
      const filename = storagePath.key.split('/').pop() || 'artifact';
      res.setHeader('Content-Disposition', `attachment; filename="${filename}"`);

      await fetchAndStreamArtifact(
        req,
        res,
        storagePath,
        sessionInfo,
        namespace,
        0, // No peek limit for download
        options,
        false,
      );
    } catch (error: any) {
      handleArtifactError(res, error);
    }
  };
}

/**
 * Handler for GET /artifacts/:artifactId/view
 * Returns full artifact content for viewing in browser.
 */
export function getArtifactViewHandler({ options }: ArtifactPreviewHandlerOptions): Handler {
  const apiServerAddress = getAddress(options.pipeline);
  const serverNamespace = options.server.serverNamespace;

  return async (req: Request, res: Response) => {
    const { artifactId } = req.params;

    if (!artifactId) {
      res.status(400).json({ error: 'Artifact ID is required' });
      return;
    }

    try {
      const artifact = await fetchArtifactFromBackend(
        artifactId,
        req,
        apiServerAddress,
        options.auth,
      );

      if (!artifact || !artifact.uri) {
        res.status(404).json({ error: 'Artifact not found or has no URI' });
        return;
      }

      const storagePath = parseStoragePath(artifact.uri);
      if (!storagePath) {
        res.status(400).json({ error: `Invalid artifact URI: ${artifact.uri}` });
        return;
      }

      const namespace = artifact.namespace || serverNamespace;
      const sessionInfo = await getStoreSessionInfo(artifact.uri, namespace);

      await fetchAndStreamArtifact(
        req,
        res,
        storagePath,
        sessionInfo,
        namespace,
        0, // No peek limit for view
        options,
        true, // Try to extract for view
      );
    } catch (error: any) {
      handleArtifactError(res, error);
    }
  };
}

function handleArtifactError(res: Response, error: any): void {
  console.error('Error fetching artifact:', error);

  if (error.status === 401) {
    res.status(401).json({
      error: 'Authentication required',
      details: 'You must be logged in to access this artifact',
    });
    return;
  }

  if (error.status === 403) {
    res.status(403).json({
      error: 'Access denied',
      details: 'You do not have permission to access this artifact',
    });
    return;
  }

  if (error.status === 404) {
    res.status(404).json({
      error: 'Artifact not found',
      details: error.message,
    });
    return;
  }

  res.status(500).json({
    error: 'Failed to fetch artifact',
    details: error.message,
  });
}

/**
 * Fetches artifact content from storage and streams it to the response.
 */
async function fetchAndStreamArtifact(
  req: Request,
  res: Response,
  storagePath: StoragePath,
  sessionInfo: SessionInfo | null,
  namespace: string,
  peek: number,
  options: UIConfigs,
  tryExtract: boolean,
): Promise<void> {
  const { source, bucket, key } = storagePath;

  console.log(`Fetching artifact from ${source}://${bucket}/${key} with peek=${peek}`);

  switch (source) {
    case 'gcs':
      await handleGCSArtifact(res, bucket, key, sessionInfo, namespace, peek);
      break;
    case 'minio':
      await handleMinioArtifact(res, bucket, key, sessionInfo, namespace, peek, options, tryExtract);
      break;
    case 's3':
      await handleS3Artifact(res, bucket, key, sessionInfo, namespace, peek, options);
      break;
    case 'http':
    case 'https':
      await handleHttpArtifact(res, source, bucket, key, peek, options);
      break;
    case 'volume':
      res.status(400).json({ error: 'Volume artifacts not supported via artifact ID endpoint' });
      break;
    default:
      res.status(400).json({ error: `Unsupported storage source: ${source}` });
  }
}

async function handleGCSArtifact(
  res: Response,
  bucket: string,
  key: string,
  sessionInfo: SessionInfo | null,
  namespace: string,
  peek: number,
): Promise<void> {
  try {
    let storageOptions: StorageOptions | undefined;

    if (sessionInfo && sessionInfo.params.fromEnv === 'false') {
      const secretName = sessionInfo.params.secretName;
      const tokenKey = sessionInfo.params.tokenKey;

      if (secretName && tokenKey) {
        const tokenString = await getK8sSecret(secretName, tokenKey, namespace);
        const credentials = parseJSONString<CredentialBody>(tokenString);
        storageOptions = {
          credentials,
          scopes: 'https://www.googleapis.com/auth/devstorage.read_write',
        };
      }
    }

    const storage = new Storage(storageOptions);
    const file = storage.bucket(bucket).file(key);

    if (peek) {
      file
        .createReadStream()
        .on('error', err => {
          console.error('GCS read error:', err);
          res.status(500).send(`Failed to read GCS file: ${err}`);
        })
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } else {
      file
        .createReadStream()
        .on('error', err => {
          console.error('GCS read error:', err);
          res.status(500).send(`Failed to read GCS file: ${err}`);
        })
        .pipe(res);
    }
  } catch (err) {
    console.error('GCS artifact error:', err);
    res.status(500).send(`Failed to download GCS file: ${err}`);
  }
}

async function handleMinioArtifact(
  res: Response,
  bucket: string,
  key: string,
  sessionInfo: SessionInfo | null,
  namespace: string,
  peek: number,
  options: UIConfigs,
  tryExtract: boolean,
): Promise<void> {
  try {
    // Build provider info for createMinioClient
    const providerInfo = sessionInfo ? buildProviderInfoString(sessionInfo) : '';

    const client = await createMinioClient(options.artifacts.minio, 'minio', providerInfo, namespace);

    const stream = await getObjectStream({ bucket, key, client, tryExtract });

    if (peek) {
      stream
        .on('error', err => {
          console.error('Minio read error:', err);
          res.status(500).send(`Failed to read Minio object: ${err}`);
        })
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } else {
      stream
        .on('error', err => {
          console.error('Minio read error:', err);
          res.status(500).send(`Failed to read Minio object: ${err}`);
        })
        .pipe(res);
    }
  } catch (err) {
    console.error('Minio artifact error:', err);
    res.status(500).send(`Failed to download Minio object: ${err}`);
  }
}

async function handleS3Artifact(
  res: Response,
  bucket: string,
  key: string,
  sessionInfo: SessionInfo | null,
  namespace: string,
  peek: number,
  options: UIConfigs,
): Promise<void> {
  try {
    // Build provider info for createMinioClient
    const providerInfo = sessionInfo ? buildProviderInfoString(sessionInfo) : '';

    const client = await createMinioClient(options.artifacts.aws, 's3', providerInfo, namespace);

    const stream = await getObjectStream({ bucket, key, client });

    if (peek) {
      stream
        .on('error', err => {
          console.error('S3 read error:', err);
          res.status(500).send(`Failed to read S3 object: ${err}`);
        })
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } else {
      stream
        .on('error', err => {
          console.error('S3 read error:', err);
          res.status(500).send(`Failed to read S3 object: ${err}`);
        })
        .pipe(res);
    }
  } catch (err) {
    console.error('S3 artifact error:', err);
    res.status(500).send(`Failed to download S3 object: ${err}`);
  }
}

async function handleHttpArtifact(
  res: Response,
  source: 'http' | 'https',
  host: string,
  path: string,
  peek: number,
  options: UIConfigs,
): Promise<void> {
  const url = `${source}://${host}/${path}`;

  try {
    const response = await fetch(url);

    if (!response.ok) {
      res.status(response.status).send(`Failed to fetch HTTP artifact: ${response.statusText}`);
      return;
    }

    if (peek) {
      response.body
        .on('error', err => res.status(500).send(`Failed to read HTTP artifact: ${err}`))
        .pipe(new PreviewStream({ peek }))
        .pipe(res);
    } else {
      response.body
        .on('error', err => res.status(500).send(`Failed to read HTTP artifact: ${err}`))
        .pipe(res);
    }
  } catch (err) {
    console.error('HTTP artifact error:', err);
    res.status(500).send(`Failed to download HTTP artifact: ${err}`);
  }
}

/**
 * Builds a provider info JSON string from SessionInfo for use with createMinioClient.
 */
function buildProviderInfoString(sessionInfo: SessionInfo): string {
  const providerInfo = {
    Provider: sessionInfo.provider,
    Params: sessionInfo.params,
  };
  return JSON.stringify(providerInfo);
}
