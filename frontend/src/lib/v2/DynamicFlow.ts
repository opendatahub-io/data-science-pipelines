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
import {Elements, FlowElement, Node} from 'react-flow-renderer';
import {
  ArtifactFlowElementData,
  ArtifactIconState,
  FlowElementDataBase,
  TaskFlowElementData,
} from 'src/components/graph/Constants';
import {PipelineSpec, PipelineTaskSpec} from 'src/generated/pipeline_spec';
import {
  buildDag,
  buildGraphLayout,
  NodeTypeNames,
  PipelineFlowElement,
} from 'src/lib/v2/StaticFlow';
import {PipelineTaskDetailTaskType, V2beta1PipelineTaskDetail, V2beta1Run, V2beta1Artifact} from "../../apisv2beta1/run";
import {logger, createScopeToTaskMap, createTaskIDToTaskMap} from "../Utils";

export const TASK_NAME_KEY = 'task_name';

export function convertSubDagToRuntimeFlowElements(
  spec: PipelineSpec,
  layers: string[],
  run: V2beta1Run,
): Elements {
  let componentSpec = spec.root;

  if (!componentSpec) {
    throw new Error('root not found in pipeline spec.');
  }

  let scopePathToTasksMap = createScopeToTaskMap(run)
  const scopeKey = layers.join(".")
  const scopeTask = scopePathToTasksMap.get(scopeKey)
  if (!scopeTask) {
    throw new Error("Scope task not found for scope key: " + scopeKey);
  }

  // Retrieve the component spec for this layer
  const componentsMap = spec.components;
  for (let index = 1; index < layers.length; index++) {
    if (layers[index].indexOf('.') <= 0) {
      // Regular layer. This layer is not an iteration of ParallelFor SubDAG.
      const tasksMap = componentSpec.dag?.tasks || {};
      const pipelineTaskSpec: PipelineTaskSpec = tasksMap[layers[index]];
      const componetRef = pipelineTaskSpec.componentRef;
      const componentName = componetRef?.name;
      if (!componentName) {
        throw new Error(
          'Unable to find the component reference for task name: ' +
            pipelineTaskSpec.taskInfo?.name || 'Task name unknown',
        );
      }
      componentSpec = componentsMap[componentName];
      if (!componentSpec) {
        throw new Error('Component not found in pipeline spec. Component name: ' + componentName);
      }
    }
  }

  if (scopeTask?.type === PipelineTaskDetailTaskType.LOOP) {
    if (!componentSpec.dag) {
      throw new Error("ParallelFor dag not found in pipeline spec.");
    }
    const iterationCountStr = scopeTask.type_attributes?.iteration_count;
    if (iterationCountStr === undefined || !iterationCountStr) {
      throw new Error("iteration count does not exist for parallelFor Execution");
    }
    const iterationCount = Number(iterationCountStr);
    if (Number.isNaN(iterationCount)) {
      throw new Error("iteration count was not a number");
    }

    // Draw sub dag nodes equal to the number of iteration_count
    let flowGraph: FlowElement[] = [];
    for (let index = 0; index < iterationCount; index++) {
      flowGraph = [...buildDag(spec, componentSpec, index), ...flowGraph];
    }
    return buildGraphLayout(flowGraph);
  }
  return buildDag(spec, componentSpec);
}

export function updateFlowElementsState(
  layers: string[],
  elems: PipelineFlowElement[],
  scopePathToTasksMap: Map<string, V2beta1PipelineTaskDetail>,
): PipelineFlowElement[] {
  let flowGraph: PipelineFlowElement[] = [];

  const scopeKey = layers.join(".")
  const dagTask = scopePathToTasksMap.get(scopeKey)
  if (!dagTask) {
    throw new Error("Scope task not found for scope key: " + scopeKey);
  }

  for (let elem of elems) {
    let updatedElem = Object.assign({}, elem);
    if (NodeTypeNames.EXECUTION === elem.type || NodeTypeNames.SUB_DAG === elem.type) {
      const taskNodeKey = elem.data?.taskKey;
      const scopeKey = [...layers, taskNodeKey].join(".")
      const scopeTask = scopePathToTasksMap.get(scopeKey)
      if (scopeTask === undefined) {
        console.warn("Scope task not found for scope key: " + scopeKey)
      } else {
        (updatedElem.data as TaskFlowElementData).state = scopeTask.state;
        (updatedElem.data as TaskFlowElementData).taskID = scopeTask.task_id;
        (updatedElem.data as TaskFlowElementData).label = scopeTask.display_name || scopeTask.name || taskNodeKey;
      }
    } else if (NodeTypeNames.ARTIFACT === elem.type) {
      const scopeKey = [...layers, updatedElem.data?.task].join(".")
      const scopeTask = scopePathToTasksMap.get(scopeKey);
      if (!scopeTask) {
        logger.error("Scope task not found for scope key: " + scopeKey);
        continue;
      }
      const taskOutputArtifacts = scopeTask.outputs?.artifacts ?? [];
      const matchingArtifactIO = taskOutputArtifacts.find(
        artifactIO => artifactIO.artifact_key === updatedElem.data?.label
      );
      if (!matchingArtifactIO?.artifacts?.length) {
        logger.error("No artifacts found for task: " + scopeKey);
        continue;
      }
      // TODO(HumairAK): Do we support list outputs in UI?
      // for now just get the first artifact.
      const artifact = matchingArtifactIO.artifacts[0];
      (updatedElem.data as ArtifactFlowElementData).artifactId = artifact.artifact_id;
      (updatedElem.data as ArtifactFlowElementData).outputArtifactKey = matchingArtifactIO.artifact_key;
      (updatedElem.data as ArtifactFlowElementData).producerTaskName = matchingArtifactIO.producer?.task_name;
      (updatedElem.data as ArtifactFlowElementData).producerTaskID = scopeTask.task_id;
      (updatedElem.data as ArtifactFlowElementData).state = ArtifactIconState.LIVE;
    }
    flowGraph.push(updatedElem);
  }
  return flowGraph;
}


export interface ArtifactWithTaskInfo {
  artifact?: V2beta1Artifact; // TODO(Humair): Or we can just store the artifact IO for each node, it has more info we can display.
  producerTaskName?: string;
  producerTaskID?: string;
  outputArtifactKey?: string;
}

export interface NodeInfo {
  task?: V2beta1PipelineTaskDetail;
  artifactWithTaskInfo?: ArtifactWithTaskInfo;
}

export function getNodeInfo(
  elem: FlowElement<FlowElementDataBase> | null,
  run: V2beta1Run,
): NodeInfo {
  if (!elem) {
    return {};
  }
  const idToTask = createTaskIDToTaskMap(run)
  if (NodeTypeNames.ARTIFACT === elem.type) {
    const artifactElem = elem as ArtifactFlowElementData;
    const producerTaskName = artifactElem.producerTaskName;
    const producerTaskID = artifactElem.producerTaskID;
    const outputArtifactKey = artifactElem.outputArtifactKey;
    if (!producerTaskName || !producerTaskID || !outputArtifactKey) {
      throw new Error("Producer task name, output artifact key, or ID not found for artifact: " + artifactElem.label);
    }
    const producerTask = idToTask.get(producerTaskID)
    if (!producerTask) {
      throw new Error("Producer task not found for task ID: " + producerTaskID);
    }
    const artifact = producerTask.outputs?.artifacts
      ?.flatMap(io => io.artifacts ?? [])
      .find(a => a.artifact_id === artifactElem.artifactId);
    if (!artifact) {
      throw new Error("Artifact not found for producer task: " + producerTaskName);
    }
    const artifactDetails = {
      artifact: artifact,
      producerTaskName: producerTaskName,
      producerTaskID: producerTaskID,
      outputArtifactKey: outputArtifactKey
    };
    return { artifactWithTaskInfo: artifactDetails};
  }
  // If not an artifact then it's a task
  const taskElem = elem as TaskFlowElementData
  if (!taskElem.taskID) {
    throw new Error("Task ID not found for task: " + taskElem.label);
  }
  const task = idToTask.get(taskElem.taskID)
  if (!task) {
    throw new Error("Task not found for task ID: " + taskElem.taskID);
  }
  return { task: task };

}
