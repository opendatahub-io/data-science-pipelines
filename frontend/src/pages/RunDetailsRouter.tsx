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
import * as JsYaml from 'js-yaml';
import { useQuery } from 'react-query';
import CircularProgress from '@material-ui/core/CircularProgress';
import { V2beta1Run } from 'src/apisv2beta1/run';
import { RouteParams } from 'src/components/Router';
import { Apis } from 'src/lib/Apis';
import { commonCss } from 'src/Css';
import { RunDetailsV2 } from 'src/pages/RunDetailsV2';

// This is a router to determine whether to show V1 or V2 run detail page.
// Note: V1 pipelines are no longer supported - all runs use V2 RunDetailsV2.
export default function RunDetailsRouter(props: any) {
  const runId = props.match.params[RouteParams.runId];
  let pipelineManifest: string | undefined;

  // Retrieves v2 run detail.
  const {
    isSuccess: getV2RunSuccess,
    isLoading: runIsLoading,
    data: v2Run,
  } = useQuery<V2beta1Run, Error>({
    queryKey: ['v2_run_detail', { id: runId }],
    queryFn: () =>
      Apis.runServiceApiV2.runServiceGetRun(runId, undefined, 'FULL'),
  });

  if (getV2RunSuccess && v2Run && v2Run.pipeline_spec) {
    pipelineManifest = JsYaml.safeDump(v2Run.pipeline_spec);
  }

  const pipelineId = v2Run?.pipeline_version_reference?.pipeline_id;
  const pipelineVersionId = v2Run?.pipeline_version_reference?.pipeline_version_id;

  const { isLoading: templateStrIsLoading, data: templateStrFromPipelineVersion } = useQuery<
    string,
    Error
  >(
    ['PipelineVersionTemplate', { pipelineId, pipelineVersionId }],
    async () => {
      if (!pipelineId || !pipelineVersionId) {
        return '';
      }
      const pipelineVersion = await Apis.pipelineServiceApiV2.pipelineServiceGetPipelineVersion(
        pipelineId,
        pipelineVersionId,
      );
      const pipelineSpec = pipelineVersion.pipeline_spec;
      return pipelineSpec ? JsYaml.safeDump(pipelineSpec) : '';
    },
    { enabled: !!pipelineVersionId, staleTime: Infinity, cacheTime: Infinity },
  );

  const templateString = pipelineManifest ?? templateStrFromPipelineVersion;

  // Show loading state only on initial load, not during background refetches
  if (runIsLoading || templateStrIsLoading) {
    return (
      <div className={commonCss.page}>
        <CircularProgress className={commonCss.absoluteCenter} />
      </div>
    );
  }

  // Show V2 run details page
  if (getV2RunSuccess && v2Run && templateString) {
    return <RunDetailsV2 pipeline_job={templateString} run={v2Run} {...props} />;
  }

  // If we couldn't get the run or template, show an error
  return (
    <div className={commonCss.page}>
      <div className={commonCss.absoluteCenter}>
        Unable to load run details. The run may not exist or the pipeline spec may be missing.
      </div>
    </div>
  );
}
