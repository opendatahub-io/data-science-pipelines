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

import * as React from 'react';
import { ErrorBoundary } from 'src/atoms/ErrorBoundary';
import { commonCss, padding } from 'src/Css';
import Banner from '../Banner';
import { MetricsVisualizations } from '../viewers/MetricsVisualizations';
import { ExecutionTitle } from './ExecutionTitle';
import { V2beta1PipelineTaskDetail, PipelineTaskDetailTaskState } from '../../apisv2beta1/run';
import { ArtifactWithTaskInfo } from '../../lib/v2/DynamicFlow';

// New V2beta1 interface
type MetricsTabPropsNew = {
  task: V2beta1PipelineTaskDetail;
  artifactDetails?: ArtifactWithTaskInfo[];
  namespace: string | undefined;
};

// Legacy MLMD interface for backward compatibility
type MetricsTabPropsLegacy = {
  execution: any; // MLMD Execution type
  namespace: string | undefined;
};

export type MetricsTabProps = MetricsTabPropsNew | MetricsTabPropsLegacy;

function isNewInterface(props: MetricsTabProps): props is MetricsTabPropsNew {
  return 'task' in props;
}

/**
 * Metrics tab renders metrics for the artifact of given execution.
 * Some system metrics are: Confusion Matrix, ROC Curve, Scalar, etc.
 * Detail can be found in https://github.com/kubeflow/pipelines/blob/master/sdk/python/kfp/dsl/io_types.py
 * Note that these metrics are only available on KFP v2 mode.
 */
export function MetricsTab(props: MetricsTabProps) {
  const { namespace } = props;

  // Handle legacy MLMD interface
  if (!isNewInterface(props)) {
    // Legacy MLMD Execution type is not supported
    // TODO(HumairAK): Re-implement MLMD Execution support or remove after full migration
    return (
      <ErrorBoundary>
        <div className={commonCss.page}>
          <div className={padding(20)}>
            <Banner
              message='Metrics visualization is temporarily unavailable during API migration.'
              mode='info'
            />
          </div>
        </div>
      </ErrorBoundary>
    );
  }

  // New V2beta1 interface
  const { task, artifactDetails } = props;
  let taskCompleted = false;
  const taskState = task.state;
  if (
    !(
      taskState === PipelineTaskDetailTaskState.RUNTIMESTATEUNSPECIFIED ||
      taskState === PipelineTaskDetailTaskState.RUNNING
    )
  ) {
    taskCompleted = true;
  }

  const taskStateUnknown = taskState === PipelineTaskDetailTaskState.RUNTIMESTATEUNSPECIFIED;

  return (
    <ErrorBoundary>
      <div className={commonCss.page}>
        <div className={padding(20)}>
          <ExecutionTitle task={task} />
          {taskStateUnknown && <Banner message='Task is in unknown state.' mode='info' />}
          {!taskStateUnknown && !taskCompleted && (
            <Banner message='Task has not completed.' mode='info' />
          )}
          {taskCompleted && (
            <MetricsVisualizations
              artifactDetails={artifactDetails || []}
              execution={task}
              namespace={namespace}
            />
          )}
        </div>
      </div>
    </ErrorBoundary>
  );
}
