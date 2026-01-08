/*
 * Copyright 2019 The Kubeflow Authors
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
import * as React from 'react';
import { stylesheet } from 'typestyle';
import { color, commonCss } from '../Css';
import { ArtifactLink } from './ArtifactLink';
import {PipelineTaskDetailTaskState, V2beta1PipelineTaskDetail} from "../apisv2beta1/run";
import {V2beta1Artifact} from "../apisv2beta1/artifact";

export const css = stylesheet({
  field: {
    flexBasis: '300px',
    marginBottom: '32px',
    padding: '4px',
  },
  resourceInfo: {
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
  },
  term: {
    color: color.grey,
    fontSize: '12px',
    letterSpacing: '0.2px',
    lineHeight: '16px',
  },
  value: {
    color: color.secondaryText,
    fontSize: '14px',
    letterSpacing: '0.2px',
    lineHeight: '20px',
  },
});

export enum ResourceType {
  ARTIFACT = 'ARTIFACT',
  TASK = 'TASK',
  // EXECUTION is kept as an alias for TASK for backward compatibility during MLMD removal
  EXECUTION = 'TASK',
}

export interface ArtifactProps {
  resourceType: ResourceType.ARTIFACT;
  resource: V2beta1Artifact | any; // 'any' for backward compatibility with MLMD types during migration
  typeName: string;
}

export interface TaskProps {
  resourceType: ResourceType.TASK;
  resource: V2beta1PipelineTaskDetail | any; // 'any' for backward compatibility with MLMD types during migration
  typeName: string;
}

// Legacy MLMD types compatibility - EXECUTION uses the same structure as TASK
export interface ExecutionProps {
  resourceType: ResourceType.EXECUTION;
  resource: any;
  typeName: string;
}

export type ResourceInfoProps = ArtifactProps | TaskProps | ExecutionProps;

export class ResourceInfo extends React.Component<ResourceInfoProps, {}> {
  public render(): JSX.Element {
    let resourceTitle = this.props.typeName;
    const stateText = getResourceStateText(this.props);
    if (stateText) {
      resourceTitle = `${resourceTitle} (${stateText})`;
    }

    // Get metadata/properties based on resource type
    const metadata = this.getMetadata();
    const metadataEntries = Object.entries(metadata);

    return (
      <section>
        <h1 className={commonCss.header}>{resourceTitle}</h1>
        {(() => {
          if (this.props.resourceType === ResourceType.ARTIFACT) {
            // Handle both MLMD artifacts (getUri method) and V2beta1 artifacts (uri property)
            const artifactUri = typeof this.props.resource?.getUri === 'function'
              ? this.props.resource.getUri()
              : this.props.resource?.uri;
            return (
              <>
                <dt className={css.term}>URI</dt>
                <dd className={css.value}>
                  <ArtifactLink artifactUri={artifactUri} />
                </dd>
              </>
            );
          }
          return null;
        })()}
        {metadataEntries.length > 0 && (
          <>
            <h2 className={commonCss.header2}>Properties</h2>
            <dl className={css.resourceInfo}>
              {metadataEntries
                .filter(([key]) => key !== '__ALL_META__')
                .map(([key, value]) => (
                  <div className={css.field} key={key} data-testid='resource-info-property'>
                    <dt className={css.term} data-testid='resource-info-property-key'>
                      {key}
                    </dt>
                    <dd className={css.value} data-testid='resource-info-property-value'>
                      {prettyPrintValue(value)}
                    </dd>
                  </div>
                ))}
            </dl>
          </>
        )}
      </section>
    );
  }

  private getMetadata(): { [key: string]: any } {
    const resource = this.props.resource;

    // Handle MLMD types (have getPropertiesMap method)
    if (typeof resource?.getPropertiesMap === 'function') {
      const metadata: { [key: string]: any } = {};
      try {
        const propsMap = resource.getPropertiesMap();
        propsMap.forEach((value: any, key: string) => {
          metadata[key] = this.getMetadataValue(value);
        });
        const customPropsMap = resource.getCustomPropertiesMap();
        customPropsMap.forEach((value: any, key: string) => {
          metadata[key] = this.getMetadataValue(value);
        });
      } catch {
        // Ignore errors when accessing MLMD maps
      }
      return metadata;
    }

    if (this.props.resourceType === ResourceType.ARTIFACT) {
      return resource.metadata || {};
    } else {
      // For tasks, combine relevant properties into a metadata-like structure
      const task = resource;
      const metadata: { [key: string]: any } = {};
      if (task.display_name) metadata['Display Name'] = task.display_name;
      if (task.name) metadata['Name'] = task.name;
      if (task.task_id) metadata['Task ID'] = task.task_id;
      if (task.run_id) metadata['Run ID'] = task.run_id;
      if (task.create_time) metadata['Created At'] = new Date(task.create_time).toString();
      if (task.end_time) metadata['Finished At'] = new Date(task.end_time).toString();
      if (task.type) metadata['Type'] = task.type;
      if (task.status_metadata?.message) metadata['Status Message'] = task.status_metadata.message;
      // Include custom properties from status_metadata if available
      if (task.status_metadata?.custom_properties) {
        Object.entries(task.status_metadata.custom_properties).forEach(([key, value]) => {
          metadata[key] = value;
        });
      }
      return metadata;
    }
  }

  // Helper to extract value from MLMD Value type
  private getMetadataValue(value: any): any {
    if (!value) return null;
    if (typeof value.getStringValue === 'function' && value.getStringValue()) {
      return value.getStringValue();
    }
    if (typeof value.getIntValue === 'function' && value.getIntValue()) {
      return value.getIntValue();
    }
    if (typeof value.getDoubleValue === 'function' && value.getDoubleValue()) {
      return value.getDoubleValue();
    }
    if (typeof value.getBoolValue === 'function') {
      return value.getBoolValue();
    }
    return value;
  }
}

function prettyPrintValue(value: any): JSX.Element | number | string {
  if (value == null) {
    return '';
  }
  if (typeof value === 'string') {
    return prettyPrintJsonValue(value);
  }
  if (typeof value === 'number') {
    return value;
  }
  if (typeof value === 'boolean') {
    return value.toString();
  }
  if (typeof value === 'object') {
    return <pre>{JSON.stringify(value, null, 2)}</pre>;
  }
  return String(value);
}

function prettyPrintJsonValue(value: string): JSX.Element | string {
  try {
    const jsonValue = JSON.parse(value);
    return <pre>{JSON.stringify(jsonValue, null, 2)}</pre>;
  } catch {
    // not JSON, return directly
    return value;
  }
}

// Get text representation of resource state.
// Works for both artifact and task.
export function getResourceStateText(props: ResourceInfoProps): string | undefined {
  if (props.resourceType === ResourceType.ARTIFACT) {
    // Prior to mlmd removal, artifacts used to have state, but now they don't.
    return 'Live'
  } else {
    // type == TASK
    const state = props.resource.state;
    switch (state) {
      case PipelineTaskDetailTaskState.RUNTIMESTATEUNSPECIFIED:
        return undefined;
      case PipelineTaskDetailTaskState.RUNNING:
        return 'Running';
      case PipelineTaskDetailTaskState.SUCCEEDED:
        return 'Succeeded';
      case PipelineTaskDetailTaskState.SKIPPED:
        return 'Skipped';
      case PipelineTaskDetailTaskState.FAILED:
        return 'Failed';
      case PipelineTaskDetailTaskState.CACHED:
        return 'Cached';
      default:
        return undefined;
    }
  }
}
