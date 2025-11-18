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

import {ComponentMeta, ComponentStory} from '@storybook/react';
import React from 'react';
import ReactFlow, {Background, Controls, MiniMap, OnLoadParams, ReactFlowProvider,} from 'react-flow-renderer';
import 'src/build/tailwind.output.css';
import {
  ArtifactFlowElementData,
  ArtifactIconState,
  FlowElementDataBase,
  TaskFlowElementData,
} from 'src/components/graph/Constants';
import {NODE_TYPES, NodeTypeNames} from 'src/lib/v2/StaticFlow';
import {PipelineTaskDetailTaskState} from "../../apisv2beta1/run";

const elements = [
  {
    id: '2',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 100 },
    data: { label: 'Default task node' } as TaskFlowElementData,
  },
  {
    id: '3',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 200 },
    data: {
      label: 'UNKNOWN task node',
      state: PipelineTaskDetailTaskState.RUNTIMESTATEUNSPECIFIED,
    } as TaskFlowElementData,
  },
  {
    id: '5',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 400 },
    data: {
      label: 'RUNNING task node',
      state: PipelineTaskDetailTaskState.RUNNING,
    } as TaskFlowElementData,
  },
  {
    id: '6',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 500 },
    data: {
      label: 'COMPLETE task node',
      state: PipelineTaskDetailTaskState.SUCCEEDED,
    } as TaskFlowElementData,
  },
  {
    id: '7',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 600 },
    data: {
      label: 'CACHED task node',
      state: PipelineTaskDetailTaskState.CACHED,
    } as TaskFlowElementData,
  },
  {
    id: '9',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 800 },
    data: {
      label: 'FAILED task node',
      state: PipelineTaskDetailTaskState.FAILED,
    } as TaskFlowElementData,
  },
  {
    id: '9',
    type: NodeTypeNames.EXECUTION,
    position: { x: 100, y: 900 },
    data: {
      label: 'invalid task node',
      state: 8 as PipelineTaskDetailTaskState,
    } as TaskFlowElementData,
  },
  {
    id: '101',
    type: NodeTypeNames.ARTIFACT,
    position: { x: 400, y: 100 },
    data: {
      label: 'DEFAULT artifact node',
      state: ArtifactIconState.UNKNOWN,
    } as ArtifactFlowElementData,
  },
  {
    id: '102',
    type: NodeTypeNames.ARTIFACT,
    position: { x: 400, y: 200 },
    data: {
      label: 'LIVE artifact node',
      state: ArtifactIconState.LIVE,
    } as ArtifactFlowElementData,
  },
  {
    id: '201',
    type: NodeTypeNames.SUB_DAG,
    position: { x: 700, y: 72 },
    data: {
      label: 'Sub-DAG node',
    } as FlowElementDataBase,
  },
];

function WrappedNodeGallery({}) {
  const onLoad = (reactFlowInstance: OnLoadParams) => {
    reactFlowInstance.fitView();
  };

  return (
    <div style={{ width: '1200px', height: '1000px' }}>
      {/* // className='flex container mx-auto' */}
      <ReactFlowProvider>
        <ReactFlow
          style={{ background: '#F5F5F5' }}
          elements={elements}
          snapToGrid={true}
          nodeTypes={NODE_TYPES}
          edgeTypes={{}}
          onLoad={onLoad}
        >
          <MiniMap />
          <Controls />
          <Background />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}

export default {
  title: 'v2/NodeGallery',
  component: WrappedNodeGallery,
  argTypes: {
    backgroundColor: { control: 'color' },
  },
} as ComponentMeta<typeof WrappedNodeGallery>;

const Template: ComponentStory<typeof WrappedNodeGallery> = args => (
  <WrappedNodeGallery {...args} />
);

export const Primary = Template.bind({});
Primary.args = {};
