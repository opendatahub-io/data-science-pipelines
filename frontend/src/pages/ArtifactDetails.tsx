/*
 * Copyright 2019 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an 'AS IS' BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { V2beta1Artifact } from 'src/apisv2beta1/artifact';
import { CircularProgress } from '@material-ui/core';
import * as React from 'react';
import { classes } from 'typestyle';
import { ResourceInfo, ResourceType } from '../components/ResourceInfo';
import { RoutePage, RouteParams } from '../components/Router';
import { ToolbarProps } from '../components/Toolbar';
import { commonCss, padding } from '../Css';
import { serviceErrorToString, titleCase } from '../lib/Utils';
import { Apis } from '../lib/Apis';
import { Page, PageProps } from './Page';

interface ArtifactDetailsState {
  artifact?: V2beta1Artifact;
}

class ArtifactDetails extends Page<{}, ArtifactDetailsState> {
  private get properTypeName(): string {
    const artifactType = this.state.artifact?.type;
    if (!artifactType) {
      return '';
    }
    // Convert enum value to display name
    return titleCase(String(artifactType));
  }

  private get id(): string {
    return this.props.match.params[RouteParams.ID];
  }

  public state: ArtifactDetailsState = {};

  public async componentDidMount(): Promise<void> {
    return this.load();
  }

  public render(): JSX.Element {
    if (!this.state.artifact) {
      return (
        <div className={commonCss.page}>
          <CircularProgress className={commonCss.absoluteCenter} />
        </div>
      );
    }
    return (
      <div className={classes(commonCss.page)}>
        <div className={classes(padding(20, 'lr'))}>
          <ResourceInfo
            resourceType={ResourceType.ARTIFACT}
            typeName={this.properTypeName}
            resource={this.state.artifact}
          />
        </div>
      </div>
    );
  }

  public getInitialToolbarState(): ToolbarProps {
    return {
      actions: {},
      breadcrumbs: [{ displayName: 'Artifacts', href: RoutePage.ARTIFACTS }],
      pageTitle: `Artifact #${this.id}`,
    };
  }

  public async refresh(): Promise<void> {
    return this.load();
  }

  private load = async (): Promise<void> => {
    try {
      const artifact = await Apis.artifactServiceApi.getArtifact(this.id);

      let title = artifact.name || `Artifact #${this.id}`;
      const version = artifact.metadata?.version;
      if (version) {
        title += ` (version: ${version})`;
      }
      this.props.updateToolbar({
        pageTitle: title,
      });
      this.setState({ artifact });
    } catch (err) {
      this.showPageError(serviceErrorToString(err));
    }
  };
}

// This guarantees that each artifact renders a different <ArtifactDetails /> instance.
const EnhancedArtifactDetails = (props: PageProps) => {
  return <ArtifactDetails {...props} key={props.match.params[RouteParams.ID]} />;
};

export default EnhancedArtifactDetails;
