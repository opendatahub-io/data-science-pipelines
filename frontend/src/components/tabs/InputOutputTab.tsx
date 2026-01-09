/*
 * Copyright 2021 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import { commonCss, padding } from 'src/Css';
import { KeyValue } from 'src/lib/StaticGraphParser';
import ArtifactPreview from '../ArtifactPreview';
import Banner from '../Banner';
import DetailsTable from '../DetailsTable';
import { ExecutionTitle } from './ExecutionTitle';
import {V2beta1PipelineTaskDetail, InputOutputsIOParameter, InputOutputsIOArtifact} from "../../apisv2beta1/run";

export type ParamList = Array<KeyValue<string>>;

/**
 * Map from artifact URI to artifact ID for lookup in ArtifactPreview component.
 */
export type URIToArtifactId = Map<string, string | undefined>;

/**
 * Contains artifact params (for display) and a map to look up artifact IDs by URI.
 */
export interface ArtifactParamsWithIds {
  params: ParamList;
  artifactIdMap: URIToArtifactId;
}

// New V2beta1 interface
export interface IOTabProps {
  task: V2beta1PipelineTaskDetail;
  namespace: string | undefined;
}

// Legacy MLMD interface for backward compatibility
interface IOTabPropsLegacy {
  execution: any; // MLMD Execution type
  namespace: string | undefined;
}

type InputOutputTabProps = IOTabProps | IOTabPropsLegacy;

function isNewInterface(props: InputOutputTabProps): props is IOTabProps {
  return 'task' in props;
}

export function InputOutputTab(props: InputOutputTabProps) {
  const { namespace } = props;

  // Handle legacy MLMD interface from V1 run details page
  // TODO(HumairAK): Remove this once v1 is removed
  if (!isNewInterface(props)) {
    return (
      <ErrorBoundary>
        <div className={commonCss.page}>
          <div className={padding(20)}>
            <Banner
              message='Input/Output visualization has been removed.'
              mode='info'
            />
          </div>
        </div>
      </ErrorBoundary>
    );
  }

  // New V2beta1 interface
  const { task } = props;
  const taskId = task.task_id || 'unknown';

  // Extract input and output parameters from the task
  const inputParams = extractInputParameters(task);
  const outputParams = extractOutputParameters(task);

  // Extract input and output artifacts from the task
  const { params: inputArtifacts, artifactIdMap: inputArtifactIdMap } = extractInputArtifacts(task);
  const { params: outputArtifacts, artifactIdMap: outputArtifactIdMap } = extractOutputArtifacts(task);

  const isIoEmpty =
    inputParams.length === 0 &&
    outputParams.length === 0 &&
    inputArtifacts.length === 0 &&
    outputArtifacts.length === 0;

  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <ExecutionTitle task={task} />

          {isIoEmpty && (
            <Banner message='There is no input/output parameter or artifact.' mode='info' />
          )}

          {inputParams.length > 0 && (
            <div>
              <DetailsTable
                key={`input-parameters-${taskId}`}
                title='Input Parameters'
                fields={inputParams}
              />
            </div>
          )}

          {inputArtifacts.length > 0 && (
            <div>
              <DetailsTable<string>
                key={`input-artifacts-${taskId}`}
                title='Input Artifacts'
                fields={inputArtifacts}
                valueComponent={ArtifactPreview}
                valueComponentProps={{
                  namespace: namespace,
                  artifactIdMap: inputArtifactIdMap,
                }}
              />
            </div>
          )}

          {outputParams.length > 0 && (
            <div>
              <DetailsTable
                key={`output-parameters-${taskId}`}
                title='Output Parameters'
                fields={outputParams}
              />
            </div>
          )}

          {outputArtifacts.length > 0 && (
            <div>
              <DetailsTable<string>
                key={`output-artifacts-${taskId}`}
                title='Output Artifacts'
                fields={outputArtifacts}
                valueComponent={ArtifactPreview}
                valueComponentProps={{
                  namespace: namespace,
                  artifactIdMap: outputArtifactIdMap,
                }}
              />
            </div>
          )}
        </div>
      </div>
    </ErrorBoundary>
  );
}

export default InputOutputTab;

function extractInputParameters(task: V2beta1PipelineTaskDetail): ParamList {
  return extractParameters(task.inputs?.parameters);
}

function extractOutputParameters(task: V2beta1PipelineTaskDetail): ParamList {
  return extractParameters(task.outputs?.parameters);
}

function extractParameters(parameters?: InputOutputsIOParameter[]): ParamList {
  if (!parameters) {
    return [];
  }
  return parameters.map(param => {
    const key = param.parameter_key || 'Unknown';
    const value = param.value !== undefined ? JSON.stringify(param.value) : '-';
    return [key, value] as KeyValue<string>;
  });
}

function extractInputArtifacts(task: V2beta1PipelineTaskDetail): ArtifactParamsWithIds {
  return extractArtifacts(task.inputs?.artifacts);
}

function extractOutputArtifacts(task: V2beta1PipelineTaskDetail): ArtifactParamsWithIds {
  return extractArtifacts(task.outputs?.artifacts);
}

/**
 * Extracts artifact display information from IO artifacts.
 * Returns params (for display) and a map from URI to artifact ID (for preview API).
 */
function extractArtifacts(ioArtifacts?: InputOutputsIOArtifact[]): ArtifactParamsWithIds {
  const params: ParamList = [];
  const artifactIdMap: URIToArtifactId = new Map<string, string | undefined>();

  if (!ioArtifacts) {
    return { params, artifactIdMap };
  }

  for (const ioArtifact of ioArtifacts) {
    const artifactKey = ioArtifact.artifact_key || 'Unknown';
    const artifacts = ioArtifact.artifacts || [];

    for (const artifact of artifacts) {
      const uri = artifact.uri || '-';
      const displayName = artifact.name || artifactKey;
      const artifactId = artifact.artifact_id;

      // Map URI to artifact ID for lookup in ArtifactPreview
      artifactIdMap.set(uri, artifactId);

      params.push([displayName, uri] as KeyValue<string>);
    }
  }

  return { params, artifactIdMap };
}
