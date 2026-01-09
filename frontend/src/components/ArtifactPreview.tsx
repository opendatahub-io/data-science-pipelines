/**
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { useQuery } from 'react-query';
import { ExternalLink } from 'src/atoms/ExternalLink';
import { color } from 'src/Css';
import { Apis } from 'src/lib/Apis';
import WorkflowParser, { StoragePath } from 'src/lib/WorkflowParser';
import { stylesheet } from 'typestyle';
import Banner from './Banner';
import { ValueComponentProps } from './DetailsTable';
import { logger } from 'src/lib/Utils';
import { URIToArtifactId } from './tabs/InputOutputTab';

const css = stylesheet({
  root: {
    width: '100%',
  },
  preview: {
    maxHeight: 250,
    overflowY: 'auto',
    padding: 3,
    backgroundColor: color.lightGrey,
  },
  topDiv: {
    display: 'flex',
    justifyContent: 'space-between',
  },
  separater: {
    width: 20, // There's minimum 20px separation between URI and view button.
    display: 'inline-block',
  },
  viewLink: {
    whiteSpace: 'nowrap',
  },
});

export interface ArtifactPreviewProps extends ValueComponentProps<string> {
  artifactId?: string;
  artifactIdMap?: URIToArtifactId;
  namespace?: string;
  maxbytes?: number;
  maxlines?: number;
}

/**
 * A component that renders a preview to an artifact with a link to the full content.
 * Uses the artifact ID-based API which handles authorization through the backend.
 */
const ArtifactPreview: React.FC<ArtifactPreviewProps> = ({
  value,
  artifactId: directArtifactId,
  artifactIdMap,
  maxbytes = 255,
  maxlines = 20,
}) => {
  // Parse storage path from URI for display purposes
  let storage: StoragePath | undefined;
  if (value) {
    try {
      storage = WorkflowParser.parseStoragePath(value);
    } catch (error) {
      logger.error(error);
    }
  }

  // Resolve artifact ID: prefer direct prop, then look up from map using URI
  const artifactId = directArtifactId || (value && artifactIdMap?.get(value)) || undefined;

  const { isSuccess, isError, data, error } = useQuery<string, Error>(
    ['artifact_preview', { artifactId, maxbytes, maxlines }],
    () => getPreview(artifactId!, maxbytes, maxlines),
    {
      staleTime: Infinity,
      enabled: !!artifactId,
    },
  );

  // Determine the link text to display
  const linkText = storage ? Apis.buildArtifactLinkText(storage) : (value || artifactId || 'Artifact');

  // If no artifact ID available, show info banner
  if (!artifactId) {
    return (
      <Banner message={'Artifact ID not available for preview: ' + value} mode='info' />
    );
  }

  // Build URLs using artifact ID
  const artifactDownloadUrl = Apis.buildArtifactDownloadUrlById(artifactId);
  const artifactViewUrl = Apis.buildArtifactViewUrlById(artifactId);

  return (
    <div className={css.root}>
      <div className={css.topDiv}>
        <ExternalLink download href={artifactDownloadUrl} title={linkText}>
          {linkText}
        </ExternalLink>
        <span className={css.separater} />
        <ExternalLink href={artifactViewUrl} className={css.viewLink}>
          View All
        </ExternalLink>
      </div>
      {isError && (
        <Banner
          message='Error in retrieving artifact preview.'
          mode='error'
          additionalInfo={getErrorMessage(error)}
        />
      )}
      {isSuccess && data && (
        <div className={css.preview}>
          <small>
            <pre>{data}</pre>
          </small>
        </div>
      )}
    </div>
  );
};

export default ArtifactPreview;

/**
 * Fetches artifact preview using the artifact ID-based API.
 */
async function getPreview(
  artifactId: string,
  maxbytes: number,
  maxlines?: number,
): Promise<string> {
  let data = await Apis.getArtifactPreview({
    artifactId,
    maxBytes: maxbytes,
    maxLines: maxlines,
  });

  // Process preview data
  if (data.length <= maxbytes && (!maxlines || data.split('\n').length < maxlines)) {
    return data;
  }

  // Truncate and add ellipsis
  data = data.slice(0, maxbytes);
  if (maxlines) {
    data = data
      .split('\n')
      .slice(0, maxlines)
      .join('\n')
      .trim();
  }
  return `${data}\n...`;
}

/**
 * Maps error types to user-friendly messages.
 */
function getErrorMessage(error: Error | null): string {
  if (!error) {
    return 'No error message';
  }

  const message = error.message.toLowerCase();

  if (message.includes('401') || message.includes('authentication')) {
    return 'You must be logged in to view this artifact.';
  }
  if (message.includes('403') || message.includes('access denied') || message.includes('permission')) {
    return 'You do not have permission to view this artifact.';
  }
  if (message.includes('404') || message.includes('not found')) {
    return 'Artifact not found. It may have been deleted.';
  }

  return error.message;
}
