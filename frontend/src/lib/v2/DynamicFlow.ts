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
import { Elements, FlowElement, Node } from 'react-flow-renderer';
import {
  ArtifactFlowElementData,
  TaskFlowElementData,
  FlowElementDataBase,
  ArtifactIconState,
} from 'src/components/graph/Constants';
import { PipelineSpec, PipelineTaskSpec } from 'src/generated/pipeline_spec';
import {
  buildDag,
  buildGraphLayout,
  getArtifactNodeKey,
  getIterationIdFromNodeKey,
  getTaskKeyFromNodeKey,
  getTaskNodeKey,
  NodeTypeNames,
  PipelineFlowElement, TASK_NODE_KEY_PREFIX,
  TaskType,
} from 'src/lib/v2/StaticFlow';
import { getArtifactNameFromEvent, LinkedArtifact, ExecutionHelpers } from 'src/mlmd/MlmdUtils';
import { NodeMlmdInfo } from 'src/pages/RunDetailsV2';
import { Artifact, Event, Execution, Value } from 'src/third_party/mlmd';
import {
  PipelineTaskDetailTaskPodType,
  V2beta1Artifact,
  V2beta1PipelineTaskDetail,
  V2beta1Run
} from "../../apisv2beta1/run";
import {logger} from "../Utils";
import ArtifactsIcon from "@material-ui/icons/BubbleChart";

export const TASK_NAME_KEY = 'task_name';
export const PARENT_DAG_ID_KEY = 'parent_dag_id';
export const ITERATION_COUNT_KEY = 'iteration_count';
export const ITERATION_INDEX_KEY = 'iteration_index';

export function convertSubDagToRuntimeFlowElements(
  spec: PipelineSpec,
  layers: string[],
  run: V2beta1Run,
): Elements {
  let componentSpec = spec.root;
  if (!componentSpec) {
    throw new Error('root not found in pipeline spec.');
  }


  let isParallelForRootDag = false;
  const componentsMap = spec.components;
  for (let index = 1; index < layers.length; index++) {
    // isParallelForRootDag = canvasIsParallelForDag(executionLayers, layers);

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

  if (isParallelForRootDag) {
    // return buildParallelForDag(executionLayers[executionLayers.length - 1]);
    // draw subdag nodes equal to the number of iteration_count
  }
  return buildDag(spec, componentSpec);
}
function canvasIsParallelForDag(executionLayers: Execution[], layers: string[]) {
  return (
    executionLayers.length === layers.length &&
    executionLayers[executionLayers.length - 1].getCustomPropertiesMap().has(ITERATION_COUNT_KEY)
  );
}

function getExecutionLayers(layers: string[], executions: Execution[]) {
  let exectuionLayers: Execution[] = [];
  if (layers.length <= 0) {
    return exectuionLayers;
  }
  const taskNameToExecution = getTaskNameToExecution(executions);

  // Get the root execution which doesn't have a task_name.
  const rootExecutions = taskNameToExecution.get('');
  if (!rootExecutions) {
    return exectuionLayers;
  }
  exectuionLayers.push(rootExecutions[0]);

  for (let index = 1; index < layers.length; index++) {
    const parentExecution = exectuionLayers[index - 1];
    const taskName = layers[index];

    let executions = taskNameToExecution.get(taskName) || [];
    // If this is an iteration of parrallelFor, remove the iteration index from layer name.
    if (taskName.indexOf('.') > 0) {
      const parallelForName = taskName.split('.')[0];
      executions = taskNameToExecution.get(parallelForName) || [];
    }

    executions = executions.filter(exec => {
      const customProperties = exec.getCustomPropertiesMap();
      if (!customProperties.has(PARENT_DAG_ID_KEY)) {
        return false;
      }
      const parentDagId = customProperties.get(PARENT_DAG_ID_KEY)?.getIntValue();
      if (parentExecution.getId() !== parentDagId) {
        return false;
      }
      if (taskName.indexOf('.') > 0) {
        const iterationIndex = Number(taskName.split('.')[1]);
        const executionIterationIndex = customProperties.get(ITERATION_INDEX_KEY)?.getIntValue();
        return iterationIndex === executionIterationIndex;
      }
      return true;
    });
    if (executions.length <= 0) {
      break;
    }

    exectuionLayers.push(executions[0]);
  }
  return exectuionLayers;
}

function buildParallelForDag(rootDagExecution: Execution): Elements {
  let flowGraph: FlowElement[] = [];
  addIterationNodes(rootDagExecution, flowGraph);
  return buildGraphLayout(flowGraph);
}

function addIterationNodes(rootDagExecution: Execution, flowGraph: PipelineFlowElement[]) {
  const taskName = rootDagExecution.getCustomPropertiesMap().get(TASK_NAME_KEY);
  const iterationCount = rootDagExecution.getCustomPropertiesMap().get(ITERATION_COUNT_KEY);
  if (taskName === undefined || !taskName.getStringValue()) {
    console.warn("Task name for the parallelFor Execution doesn't exist.");
    return;
  }
  if (iterationCount === undefined || !iterationCount.getIntValue()) {
    console.warn("Iteration Count for the parallelFor Execution doesn't exist.");
    return;
  }

  const taskNameStr = taskName.getStringValue();
  const iterationCountInt = iterationCount.getIntValue();
  for (let index = 0; index < iterationCountInt; index++) {
    const iterationNodeName = `${taskNameStr}.${index}`;
    // One iteration is a sub-DAG instance.
    const node: Node<FlowElementDataBase> = {
      id: getTaskNodeKey(iterationNodeName),
      data: { label: iterationNodeName, taskType: TaskType.DAG },
      position: { x: 100, y: 200 },
      type: NodeTypeNames.SUB_DAG,
    };
    flowGraph.push(node);
  }
}

// 1. Get the Pipeline Run context using run ID (FOR subDAG, we need to wait for design)
// 2. Fetch all executions by context. Create Map for task_name => Execution
// 3. Fetch all Events by Context. Create Map for OUTPUT events: execution_id => Events
// 5. Fetch all Artifacts by Context.
// 6. Create Map for artifacts: artifact_id => Artifact
//    a. For each task in the flowElements, find its execution state.
//    b. For each artifact node, get its task name.
//    c. get Execution from Map, then get execution_id.
//    d. get Events from Map, then get artifact name from path.
//    e. for the Event which matches artifact name, get artifact_id.
//    f. get Artifact and update the state.

// Construct ArtifactNodeKey -> Artifact Map
//    for each OUTPUT event, get execution id and artifact id
//         get execution task_name from Execution map
//         get artifact name from Event path
//         get Artifact from Artifact map
//         set ArtifactNodeKey -> Artifact.
// Elements change to Map node key => node, edge key => edge
// For each node: (DAG execution doesn't have design yet)
//     If TASK:
//         Find exeuction from using task_name
//         Update with execution state
//     If ARTIFACT:
//         Get task_name and artifact_name
//         Get artifact from Master Map
//         Update with artifact state
//     IF SUBDAG: (Not designed)
//         similar to TASK, but needs to determine subDAG type.

// Questions:
//    How to handle DAG state?
//    How to handle subDAG input artifacts and parameters?
//    How to handle if-condition? and show the state
//    How to handle parallel-for? and list of workers.

export function updateFlowElementsState(
  layers: string[],
  elems: PipelineFlowElement[],
  scopePathToTasksMap: Map<string, V2beta1PipelineTaskDetail>,
): PipelineFlowElement[] {
  // const executionLayers = getExecutionLayers(layers, executions);
  // if (executionLayers.length < layers.length) {
  //   // This Sub DAG is not executed yet. There is no runtime information to update.
  //   return elems;
  // }
  //
  // const taskNameToExecution = getTaskNameToExecution(executions);
  // const executionIdToExectuion = getExectuionIdToExecution(executions);
  // const artifactIdToArtifact = getArtifactIdToArtifact(artifacts);
  // const artifactNodeKeyToArtifact = getArtifactNodeKeyToArtifact(
  //   events,
  //   executionIdToExectuion,
  //   artifactIdToArtifact,
  // );

  let flowGraph: PipelineFlowElement[] = [];

  // if (canvasIsParallelForDag(executionLayers, layers)) {
  //   const parallelForDagExecution = executionLayers[executionLayers.length - 1];
  //   const executions = taskNameToExecution.get(
  //     parallelForDagExecution
  //       .getCustomPropertiesMap()
  //       .get(TASK_NAME_KEY)
  //       ?.getStringValue() || parallelForDagExecution.getName(),
  //   );
  //
  //   for (let elem of elems) {
  //     let updatedElem = Object.assign({}, elem);
  //     const iterationId = Number(getIterationIdFromNodeKey(updatedElem.id));
  //     const matchedExecs = executions?.filter(exec => {
  //       const customProperties = exec.getCustomPropertiesMap();
  //       const iteration_index = customProperties.get(ITERATION_INDEX_KEY)?.getIntValue();
  //       const parent_dag_id = customProperties.get(PARENT_DAG_ID_KEY)?.getIntValue();
  //       return parent_dag_id === parallelForDagExecution.getId() && iteration_index === iterationId;
  //     });
  //     if (matchedExecs && matchedExecs.length > 0) {
  //       (updatedElem.data as SubDagFlowElementData).state = matchedExecs[0].getLastKnownState();
  //     }
  //     flowGraph.push(updatedElem);
  //   }
  //   return flowGraph;
  // }


  // Tasks in a dag scope have unique names, and artifact keys in a given task are unique to a task
  // So we build a map with keys <task_name>.<artifact_key> => Artifact for easy indexing for the
  // artifacts in this dag scope.


  for (let elem of elems) {

    let updatedElem = Object.assign({}, elem);
    if (NodeTypeNames.EXECUTION === elem.type || NodeTypeNames.SUB_DAG === elem.type) {
      const taskNodeKey = removeAnyPrefix(elem.id, TASK_NODE_KEY_PREFIX)
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

function removeAnyPrefix(str: string, prefix: string): string {
  if (str.startsWith(prefix)) {
    return str.slice(prefix.length);
  }
  return str;
}

export function getNodeTaskInfo(
  elem: FlowElement<FlowElementDataBase> | null,
  run: V2beta1Run,
): V2beta1PipelineTaskDetail {
  if (!elem) {
    return {};
  }
  // const taskNameToExecution = getTaskNameToExecution(executions);
  // const executionIdToExectuion = getExectuionIdToExecution(executions);
  // const artifactIdToArtifact = getArtifactIdToArtifact(artifacts);
  // const artifactNodeKeyToArtifact = getArtifactNodeKeyToArtifact(
  //   events,
  //   executionIdToExectuion,
  //   artifactIdToArtifact,
  // );
  //
  // if (NodeTypeNames.EXECUTION === elem.type) {
  //   const taskLabel = getTaskLabelByPipelineFlowElement(elem);
  //   const executions = taskNameToExecution
  //     .get(taskLabel)
  //     ?.filter(exec => exec.getId() === elem.data?.mlmdId);
  //   return executions ? { execution: executions[0] } : {};
  // } else if (NodeTypeNames.ARTIFACT === elem.type) {
  //   let linkedArtifact = artifactNodeKeyToArtifact.get(elem.id);
  //
  //   // Detect whether Artifact is an output of SubDAG, if so, search its source artifact.
  //   let artifactData = elem.data as ArtifactFlowElementData;
  //   if (artifactData && artifactData.outputArtifactKey && artifactData.producerSubtask) {
  //     // SubDAG output artifact has reference to inner subtask and artifact.
  //     const subArtifactKey = getArtifactNodeKey(
  //       artifactData.producerSubtask,
  //       artifactData.outputArtifactKey,
  //     );
  //     linkedArtifact = artifactNodeKeyToArtifact.get(subArtifactKey);
  //   }
  //
  //   const executionId = linkedArtifact?.event.getExecutionId();
  //   const execution = executionId ? executionIdToExectuion.get(executionId) : undefined;
  //   return { execution, linkedArtifact };
  // } else if (NodeTypeNames.SUB_DAG === elem.type) {
  //   // TODO: Update sub-dag state based on future design.
  //   const taskLabel = getTaskLabelByPipelineFlowElement(elem);
  //   const executions = taskNameToExecution
  //     .get(taskLabel)
  //     ?.filter(exec => exec.getId() === elem.data?.mlmdId);
  //   return executions ? { execution: executions[0] } : {};
  // }
    return {};
}

function getTaskNameToExecution(executions: Execution[]): Map<string, Execution[]> {
  const map = new Map<string, Execution[]>();
  for (let exec of executions) {
    const taskName = getTaskName(exec);
    if (!taskName) {
      continue;
    }
    const taskNameStr = taskName.getStringValue();
    const execs = map.get(taskNameStr);
    if (execs) {
      execs.push(exec);
    } else {
      map.set(taskNameStr, [exec]);
    }
  }
  return map;
}

function getTaskName(exec: Execution): Value | undefined {
  const customProperties = exec.getCustomPropertiesMap();
  if (!customProperties.has(TASK_NAME_KEY)) {
    console.warn("task_name key doesn't exist for custom properties of Execution " + exec.getId());
    return undefined;
  }
  const taskName = customProperties.get(TASK_NAME_KEY);
  if (!taskName) {
    console.warn(
      "task_name value doesn't exist for custom properties of Execution " + exec.getId(),
    );
    return undefined;
  }
  return taskName;
}
