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

import HelpIcon from '@material-ui/icons/Help';
import React from 'react';
import { Array as ArrayRunType, Failure, Number, Record, String, ValidationError } from 'runtypes';
import IconWithTooltip from 'src/atoms/IconWithTooltip';
import { color, padding } from 'src/Css';
import Banner from '../Banner';
import ConfusionMatrix, { ConfusionMatrixConfig } from './ConfusionMatrix';
import { HTMLViewerConfig } from './HTMLViewer';
import { MarkdownViewerConfig } from './MarkdownViewer';
import PagedTable from './PagedTable';
import { PlotType } from './Viewer';
import {
  FullArtifactPathMap,
  RocCurveColorMap,
} from 'src/lib/v2/CompareUtils';
import { ArtifactWithTaskInfo } from '../../lib/v2/DynamicFlow';
import { V2beta1PipelineTaskDetail } from '../../apisv2beta1/run';
import { V2beta1Artifact, ArtifactArtifactType } from '../../apisv2beta1/artifact';

interface MetricsVisualizationsProps {
  artifactDetails: ArtifactWithTaskInfo[];
  execution: V2beta1PipelineTaskDetail;
  namespace: string | undefined;
}

/**
 * Visualize system metrics based on artifact input. There can be multiple artifacts
 * and multiple visualizations associated with one artifact.
 *
 * TODO(HumairAK): This component has been simplified during MLMD removal.
 * Many features like ROC curves, confusion matrices, HTML/Markdown viewers
 * need to be reimplemented to work with the new V2beta1 API types.
 */
export function MetricsVisualizations({
  artifactDetails,
  execution,
  namespace,
}: MetricsVisualizationsProps) {
  // Filter artifacts by type
  const classificationMetricsArtifacts = artifactDetails.filter(
    ad => ad.artifact?.type === ArtifactArtifactType.ClassificationMetric
  );
  const metricsArtifacts = artifactDetails.filter(
    ad => ad.artifact?.type === ArtifactArtifactType.Metric
  );
  const htmlArtifacts = artifactDetails.filter(
    ad => ad.artifact?.type === ArtifactArtifactType.HTML
  );
  const mdArtifacts = artifactDetails.filter(
    ad => ad.artifact?.type === ArtifactArtifactType.Markdown
  );

  // TODO(HumairAK): Re-implement HTML and Markdown artifact downloading
  // using the new V2beta1 API types.

  if (
    classificationMetricsArtifacts.length === 0 &&
    metricsArtifacts.length === 0 &&
    htmlArtifacts.length === 0 &&
    mdArtifacts.length === 0
  ) {
    return <Banner message='There is no metrics artifact available in this step.' mode='info' />;
  }

  return (
    <>
      {/* Classification Metrics */}
      {classificationMetricsArtifacts.map(artifactDetail => {
        const artifact = artifactDetail.artifact;
        if (!artifact) return null;
        return (
          <React.Fragment key={artifact.artifact_id || artifactDetail.outputArtifactKey}>
            <ConfidenceMetricsSectionV2 artifact={artifact} />
            <ConfusionMatrixSectionV2 artifact={artifact} />
          </React.Fragment>
        );
      })}

      {/* Scalar Metrics */}
      {metricsArtifacts.map(artifactDetail => {
        const artifact = artifactDetail.artifact;
        if (!artifact) return null;
        return (
          <ScalarMetricsSectionV2
            artifact={artifact}
            key={artifact.artifact_id || artifactDetail.outputArtifactKey}
          />
        );
      })}

      {/* HTML Artifacts */}
      {htmlArtifacts.length > 0 && (
        <div className={padding(20, 'lrt')}>
          <Banner
            message='HTML visualization is temporarily unavailable during API migration.'
            mode='info'
          />
        </div>
      )}

      {/* Markdown Artifacts */}
      {mdArtifacts.length > 0 && (
        <div className={padding(20, 'lrt')}>
          <Banner
            message='Markdown visualization is temporarily unavailable during API migration.'
            mode='info'
          />
        </div>
      )}
    </>
  );
}

const ROC_CURVE_DEFINITION =
  'The receiver operating characteristic (ROC) curve shows the trade-off between true positive rate and false positive rate. ' +
  'A lower threshold results in a higher true positive rate (and a higher false positive rate), ' +
  'while a higher threshold results in a lower true positive rate (and a lower false positive rate)';

export interface ConfidenceMetricsFilter {
  selectedIds: string[];
  setSelectedIds: (selectedIds: string[]) => void;
  fullArtifactPathMap: FullArtifactPathMap;
  selectedIdColorMap: RocCurveColorMap;
  setSelectedIdColorMap: (selectedIdColorMap: RocCurveColorMap) => void;
  lineColorsStack: string[];
  setLineColorsStack: (lineColorsStack: string[]) => void;
}

export interface ConfidenceMetricsSectionProps {
  artifact: V2beta1Artifact;
  filter?: ConfidenceMetricsFilter;
}

/**
 * V2 version of ConfidenceMetricsSection using new V2beta1 API types
 */
function ConfidenceMetricsSectionV2({ artifact }: { artifact: V2beta1Artifact }) {
  const metadata = artifact.metadata || {};
  const confidenceMetrics = metadata['confidenceMetrics'];
  const name = artifact.name || 'Unknown';

  if (!confidenceMetrics) {
    return null;
  }

  // TODO(HumairAK): Re-implement ROC curve visualization with new types
  return (
    <div className={padding(40, 'lrt')}>
      <div className={padding(40, 'b')}>
        <h3>
          {'ROC Curve: ' + name}{' '}
          <IconWithTooltip
            Icon={HelpIcon}
            iconColor={color.weak}
            tooltip={ROC_CURVE_DEFINITION}
          ></IconWithTooltip>
        </h3>
      </div>
      <Banner
        message='ROC Curve visualization is temporarily unavailable during API migration.'
        mode='info'
      />
    </div>
  );
}

type AnnotationSpec = {
  displayName: string;
};
type Row = {
  row: number[];
};
type ConfusionMatrixInput = {
  annotationSpecs: AnnotationSpec[];
  rows: Row[];
};

const CONFUSION_MATRIX_DEFINITION =
  'The number of correct and incorrect predictions are ' +
  'summarized with count values and broken down by each class. ' +
  'The higher value on cell where Predicted label matches True label, ' +
  'the better prediction performance of this model is.';

/**
 * V2 version of ConfusionMatrixSection using new V2beta1 API types
 */
function ConfusionMatrixSectionV2({ artifact }: { artifact: V2beta1Artifact }) {
  const metadata = artifact.metadata || {};
  const confusionMatrix = metadata['confusionMatrix'];
  const name = artifact.name || 'Unknown';

  if (!confusionMatrix) {
    return null;
  }

  // Extract struct if present
  const matrixData = confusionMatrix.struct || confusionMatrix;

  const { error } = validateConfusionMatrix(matrixData as any);

  if (error) {
    const errorMsg = 'Error in ' + name + " artifact's confusionMatrix data format.";
    return <Banner message={errorMsg} mode='error' additionalInfo={error} />;
  }

  return (
    <div className={padding(40)}>
      <div className={padding(40, 'b')}>
        <h3>
          {'Confusion Matrix: ' + name}{' '}
          <IconWithTooltip
            Icon={HelpIcon}
            iconColor={color.weak}
            tooltip={CONFUSION_MATRIX_DEFINITION}
          ></IconWithTooltip>
        </h3>
      </div>
      <ConfusionMatrix configs={buildConfusionMatrixConfig(matrixData as any)} />
    </div>
  );
}

const ConfusionMatrixInputRunType = Record({
  annotationSpecs: ArrayRunType(
    Record({
      displayName: String,
    }),
  ),
  rows: ArrayRunType(Record({ row: ArrayRunType(Number) })),
});

function validateConfusionMatrix(input: any): { error?: string } {
  if (!input) return { error: 'confusionMatrix does not exist.' };
  try {
    const matrix = ConfusionMatrixInputRunType.check(input);
    const height = matrix.rows.length;
    const annotationLen = matrix.annotationSpecs.length;
    if (annotationLen !== height) {
      throw new ValidationError({
        message:
          'annotationSpecs has different length ' + annotationLen + ' than rows length ' + height,
      } as Failure);
    }
    for (let x of matrix.rows) {
      if (x.row.length !== height)
        throw new ValidationError({
          message: 'row: ' + JSON.stringify(x) + ' has different length of columns from rows.',
        } as Failure);
    }
  } catch (e) {
    if (e instanceof ValidationError) {
      return { error: e.message + '. Data: ' + JSON.stringify(input) };
    }
  }
  return {};
}

function buildConfusionMatrixConfig(
  confusionMatrix: ConfusionMatrixInput,
): ConfusionMatrixConfig[] {
  return [
    {
      type: PlotType.CONFUSION_MATRIX,
      axes: ['True label', 'Predicted label'],
      labels: confusionMatrix.annotationSpecs.map(annotation => annotation.displayName),
      data: confusionMatrix.rows.map(x => x.row),
    },
  ];
}

/**
 * V2 version of ScalarMetricsSection using new V2beta1 API types
 */
function ScalarMetricsSectionV2({ artifact }: { artifact: V2beta1Artifact }) {
  const metadata = artifact.metadata || {};
  const name = artifact.name || 'Unknown';

  const data = Object.entries(metadata)
    .filter(([key]) => key !== 'display_name')
    .map(([key, value]) => ({
      key,
      value: JSON.stringify(value),
    }));

  if (data.length === 0) {
    return null;
  }

  return (
    <div className={padding(40, 'lrt')}>
      <div className={padding(40, 'b')}>
        <h3>{'Scalar Metrics: ' + name}</h3>
      </div>
      <PagedTable
        configs={[
          {
            data: data.map(d => [d.key, d.value]),
            labels: ['name', 'value'],
            type: PlotType.TABLE,
          },
        ]}
      />
    </div>
  );
}

// TODO(HumairAK): The following exports are kept for backward compatibility
// but may need to be removed or updated once all consumers are migrated.

export async function getHtmlViewerConfig(
  htmlArtifacts: any[] | undefined,
  namespace: string | undefined,
): Promise<HTMLViewerConfig[]> {
  // TODO(HumairAK): Re-implement with new V2beta1 API types
  return [];
}

export async function getMarkdownViewerConfig(
  markdownArtifacts: any[] | undefined,
  namespace: string | undefined,
): Promise<MarkdownViewerConfig[]> {
  // TODO(HumairAK): Re-implement with new V2beta1 API types
  return [];
}

// Legacy exports for backward compatibility
// TODO(HumairAK): These are stubs for backward compatibility during MLMD removal.
// They need to be reimplemented with new V2beta1 API types.

export function ConfidenceMetricsSection({
  linkedArtifacts,
  filter,
}: {
  linkedArtifacts?: any[];
  filter?: ConfidenceMetricsFilter;
}) {
  return (
    <Banner
      message='ConfidenceMetricsSection is temporarily unavailable during API migration.'
      mode='info'
    />
  );
}

interface ConfusionMatrixSectionLegacyProps {
  artifact: any;
}

export function ConfusionMatrixSection({ artifact }: ConfusionMatrixSectionLegacyProps) {
  // TODO(HumairAK): This is a stub for backward compatibility.
  // The component needs to be reimplemented with new V2beta1 API types.
  return (
    <Banner
      message='ConfusionMatrixSection is temporarily unavailable during API migration.'
      mode='info'
    />
  );
}
