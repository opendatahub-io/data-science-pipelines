# Test Plan: Bring Your Own Argo Workflows (BYOAW)

## Table of Contents
1. [Overview](#overview)
2. [Test Scope](#test-scope)
3. [Test Environment Requirements](#test-environment-requirements)
4. [Test Categories](#test-categories)
5. [Test Execution Schedule](#test-execution-schedule)
6. [Success Criteria](#success-criteria)
7. [Risk Assessment](#risk-assessment)

## Overview

This test plan validates the "Bring Your Own Argo Workflows" feature, which enables Data Science Pipelines to work with existing Argo Workflows installations instead of deploying dedicated WorkflowControllers. The feature includes a global configuration mechanism to disable DSP-managed WorkflowControllers and ensures compatibility with user-provided Argo Workflows.

The plan covers comprehensive testing scenarios including:
- **Co-existence validation** of DSP and external Argo controllers competing for same events
- **Pre-existing Argo detection** and prevention mechanisms
- **CRD update-in-place** functionality and conflict resolution
- **RBAC compatibility** across different permission models (cluster vs namespace level)
- **Workflow schema version compatibility** and API compatibility validation
- **Z-stream (patch) version compatibility** testing
- **Data preservation** for WorkflowTemplates, CronWorkflows, and pipeline data
- **Independent lifecycle management** of RHOAI and external Argo installations
- **Project-level access controls** ensuring workflow visibility boundaries
- **Comprehensive migration scenarios** and upgrade path validation

## Test Scope

### In Scope
- Global configuration toggle to disable/enable WorkflowControllers across all DSPAs
- Compatibility validation with external Argo Workflows installations
- Version compatibility matrix testing (N and N-1 versions)
- Migration scenarios between DSP-managed and external Argo configurations
- Conflict detection and resolution mechanisms
- Co-existence testing of DSP and external WorkflowControllers competing for same events
- RBAC compatibility across different permission models (cluster vs namespace level)
- Workflow schema version compatibility validation
- DSPA lifecycle management with external Argo
- Security and RBAC integration with external Argo
- Performance impact assessment
- Upgrade scenarios for RHOAI with external Argo
- Hello world pipeline validation in co-existence scenarios

### Out of Scope
- Partial ArgoWF installs combined with DSP-shipped Workflow Controller
- Isolation between DSP ArgoWF WC and vanilla cluster-scoped ArgoWF installation

## Test Environment Requirements

### Prerequisites
- OpenShift/Kubernetes clusters with RHOAI/DSP installed
- Multiple test environments with different Argo Workflows versions
- Access to modify DataScienceCluster and DSPA configurations
- Sample pipelines covering various complexity levels
- Test data for migration scenarios

### Test Environments
| Environment | Argo Version | DSP Version | Purpose |
|-------------|--------------|-------------|---------|
| Env-1 | 3.4.16 | Current | N version compatibility |
| Env-2 | 3.4.15 | Current | N-1 version compatibility |
| Env-3 | 3.5.x | Current | Future compatibility testing |
| Env-4 | Various | Previous | Upgrade scenarios |

## Test Categories

## 1. Cluster Configuration Tests

### 1.1 Global Configuration Toggle

| Test Case ID | TC-CC-001 |
|---|---|
| **Test Case Summary** | Verify global toggle to disable WorkflowControllers works correctly |
| **Test Steps** | 1. Install RHOAI with default configuration (WorkflowControllers enabled)<br/>2. Create DSPA and verify WorkflowController deployment<br/>3. Update DataScienceCluster to disable WorkflowControllers:<br/>`spec.components.datasciencepipelines.argoWorkflowsControllers.managementState: Removed`<br/>4. Verify existing WorkflowControllers are removed<br/>5. Create new DSPA and verify no WorkflowController is deployed |
| **Expected Results** | - Global toggle successfully disables WorkflowController deployment<br/>- Existing WorkflowControllers are cleanly removed<br/>- New DSPAs respect global configuration<br/>- No data loss during WorkflowController removal |

| Test Case ID | TC-CC-002 |
|---|---|
| **Test Case Summary** | Verify re-enabling WorkflowControllers after global disable |
| **Test Steps** | 1. Start with globally disabled WorkflowControllers<br/>2. Create DSPA without WorkflowController<br/>3. Re-enable WorkflowControllers globally<br/>4. Verify WorkflowController is deployed to existing DSPA<br/>5. Create new DSPA and verify WorkflowController deployment |
| **Expected Results** | - Global re-enable successfully restores WorkflowController deployment<br/>- Existing DSPAs receive WorkflowControllers<br/>- New DSPAs deploy with WorkflowControllers<br/>- Pipeline history and data preserved |

### 1.2 Kubernetes Native Mode

| Test Case ID | TC-CC-003 |
|---|---|
| **Test Case Summary** | Verify BYOAW compatibility with Kubernetes Native Mode |
| **Test Steps** | 1. Configure cluster for Kubernetes Native Mode<br/>2. Install external Argo Workflows<br/>3. Disable DSP WorkflowControllers globally<br/>4. Create DSPA and execute pipelines<br/>5. Verify Kubernetes native execution with external Argo |
| **Expected Results** | - Kubernetes Native Mode works with external Argo<br/>- Pipeline execution uses Kubernetes-native constructs<br/>- No conflicts between modes |

### 1.3 FIPS Mode Compatibility

| Test Case ID | TC-CC-004 |
|---|---|
| **Test Case Summary** | Verify BYOAW works in FIPS-enabled clusters |
| **Test Steps** | 1. Configure FIPS-enabled cluster<br/>2. Install FIPS-compatible external Argo<br/>3. Configure DSPA with external Argo<br/>4. Execute pipeline suite<br/>5. Verify FIPS compliance maintained |
| **Expected Results** | - External Argo respects FIPS requirements<br/>- Pipeline execution maintains FIPS compliance<br/>- No cryptographic violations |

### 1.4 Disconnected Cluster Support

| Test Case ID | TC-CC-005 |
|---|---|
| **Test Case Summary** | Verify BYOAW functionality in disconnected environments |
| **Test Steps** | 1. Configure disconnected cluster environment<br/>2. Install external Argo from local registry<br/>3. Configure DSPA for external Argo<br/>4. Execute pipelines using local artifacts<br/>5. Verify offline operation |
| **Expected Results** | - External Argo operates in disconnected mode<br/>- Pipeline execution works without external connectivity<br/>- Local registries and artifacts accessible |

### 1.5 Platform-Level CRD and RBAC Management

| Test Case ID | TC-CC-006 |
|---|---|
| **Test Case Summary** | Verify platform-level Argo CRDs and RBAC remain intact with external Argo |
| **Test Steps** | 1. Install DSPO which creates platform-level Argo CRDs and RBAC<br/>2. Install external Argo with different CRD versions<br/>3. Toggle global WorkflowController disable<br/>4. Verify platform CRDs are not removed<br/>5. Test that user modifications to CRDs are preserved<br/>6. Verify RBAC conflicts are handled appropriately |
| **Expected Results** | - Platform-level CRDs remain intact<br/>- User CRD modifications preserved<br/>- RBAC conflicts resolved without breaking functionality<br/>- Platform operator doesn't overwrite user changes |

### 1.6 Sub-Component Removal Testing

| Test Case ID | TC-CC-007 |
|---|---|
| **Test Case Summary** | Verify sub-component removal functionality for WorkflowControllers |
| **Test Steps** | 1. Deploy DSPA with WorkflowController enabled<br/>2. Execute pipelines and accumulate run data<br/>3. Disable WorkflowController globally<br/>4. Verify WorkflowController is removed but data preserved<br/>5. Verify backing data (run details, metrics) remains intact<br/>6. Test re-enabling WorkflowController preserves historical data |
| **Expected Results** | - WorkflowController removed cleanly<br/>- Run details and metrics preserved<br/>- Historical pipeline data remains accessible<br/>- Re-enabling restores full functionality |

### 1.7 Pre-existing Argo Detection and Prevention

| Test Case ID | TC-CC-008 |
|---|---|
| **Test Case Summary** | Verify detection and prevention of DSPA creation when pre-existing Argo exists |
| **Test Steps** | 1. Install external Argo Workflows on cluster<br/>2. Install RHOAI DSP operator<br/>3. Attempt to create DSPA with default configuration (WC enabled)<br/>4. Verify detection mechanism identifies pre-existing Argo<br/>5. Test prevention of DSPA creation or automatic WC disable<br/>6. Verify appropriate warning/guidance messages<br/>7. Test manual override if supported |
| **Expected Results** | - Pre-existing Argo installation detected<br/>- DSPA creation prevented or WC automatically disabled<br/>- Clear guidance provided to user<br/>- Manual override works when applicable<br/>- No conflicts or resource competition |

### 1.8 CRD Update-in-Place Testing

| Test Case ID | TC-CC-009 |
|---|---|
| **Test Case Summary** | Verify CRD update-in-place when differences exist between pre-existing and shipped CRDs |
| **Test Steps** | 1. Install external Argo with specific CRD version<br/>2. Create Workflows, WorkflowTemplates, and CronWorkflows<br/>3. Install DSP with different compatible CRD version<br/>4. Verify CRDs are updated in-place<br/>5. Verify existing CRs (Workflows, WorkflowTemplates, CronWorkflows) remain intact<br/>6. Test new CR creation with updated CRD schema<br/>7. Verify no data loss or corruption |
| **Expected Results** | - CRDs updated in-place successfully<br/>- Existing Workflows, WorkflowTemplates, CronWorkflows preserved<br/>- New CRs work with updated schema<br/>- No data loss or corruption<br/>- Compatibility maintained |

## 2. Positive Functional Tests

### 2.1 Basic Pipeline Execution

| Test Case ID | TC-PF-001 |
|---|---|
| **Test Case Summary** | Verify basic pipeline execution with external Argo |
| **Test Steps** | 1. Configure DSPA with external Argo<br/>2. Submit simple addition pipeline<br/>3. Monitor execution through DSP UI<br/>4. Verify completion and results<br/>5. Check logs and artifacts |
| **Expected Results** | - Pipeline submits successfully<br/>- Execution progresses normally<br/>- Results accessible through DSP interface<br/>- Logs and monitoring functional |

### 2.2 Complex Pipeline Types

| Test Case ID | TC-PF-002 |
|---|---|
| **Test Case Summary** | Execute comprehensive pipeline types from valid pipeline files |
| **Test Steps** | 1. Execute pipelines from `data/pipeline_files/valid/` including:<br/>   - Pipelines with artifacts<br/>   - Pipelines without artifacts<br/>   - For loop constructs<br/>   - Parallel for execution<br/>   - Custom root KFP components<br/>   - Custom python package indexes<br/>   - Custom base images<br/>   - Pipelines with input parameters<br/>   - Pipelines without input parameters<br/>   - Pipelines with output artifacts<br/>   - Pipelines without output artifacts<br/>   - Pipelines with iteration count<br/>   - Pipelines with retry mechanisms<br/>   - Pipelines with certificate handling<br/>   - Conditional branching pipelines<br/>2. Verify each pipeline type executes correctly<br/>3. Validate artifacts, metadata, and custom configurations |
| **Expected Results** | - All pipeline types execute successfully<br/>- Custom components and packages work correctly<br/>- Retry and iteration logic functions properly<br/>- Certificate handling operates securely<br/>- Artifacts and metadata preserved correctly |

### 2.3 Pod Spec Override Testing

| Test Case ID | TC-PF-003 |
|---|---|
| **Test Case Summary** | Verify pipeline execution with Pod spec overrides |
| **Test Steps** | 1. Configure pipelines with Pod spec patches:<br/>   - Node taints and tolerations<br/>   - PVC mounts<br/>   - Custom labels and annotations<br/>   - Resource limits<br/>2. Execute pipelines with external Argo<br/>3. Verify Pod specifications applied correctly |
| **Expected Results** | - Pod spec overrides applied successfully<br/>- Pipelines schedule on correct nodes<br/>- PVCs mounted and accessible<br/>- Custom labels and annotations present |

### 2.4 Multi-DSPA Environment

| Test Case ID | TC-PF-004 |
|---|---|
| **Test Case Summary** | Verify multiple DSPAs sharing external Argo |
| **Test Steps** | 1. Create DSPAs in different namespaces<br/>2. Configure all for external Argo<br/>3. Execute pipelines simultaneously<br/>4. Verify namespace isolation<br/>5. Check resource sharing and conflicts |
| **Expected Results** | - Multiple DSPAs operate independently<br/>- Proper namespace isolation maintained<br/>- No pipeline interference or data leakage<br/>- Resource sharing works correctly |

## 3. Negative Functional Tests

### 3.1 Conflicting WorkflowController Detection

| Test Case ID | TC-NF-001 |
|---|---|
| **Test Case Summary** | Verify behavior with conflicting WorkflowController configurations |
| **Test Steps** | 1. Deploy DSPA with WorkflowController enabled<br/>2. Install external Argo on same cluster<br/>3. Attempt pipeline execution<br/>4. Document conflicts and behavior<br/>5. Test conflict resolution mechanisms |
| **Expected Results** | - System behavior is predictable<br/>- Appropriate warnings displayed<br/>- No data corruption<br/>- Clear guidance provided |

### 3.1.1 Co-existing WorkflowController Event Conflicts

| Test Case ID | TC-NF-001a |
|---|---|
| **Test Case Summary** | Test DSP and External WorkflowControllers co-existing and competing for same events |
| **Test Steps** | 1. Deploy DSPA with internal WorkflowController<br/>2. Install external Argo WorkflowController watching same namespaces<br/>3. Submit pipeline that creates Workflow CRs<br/>4. Monitor which controller processes the workflow<br/>5. Verify event handling and potential conflicts<br/>6. Test resource ownership and cleanup |
| **Expected Results** | - Event conflicts properly identified<br/>- Clear ownership of workflow resources<br/>- No orphaned or stuck workflows<br/>- Predictable controller behavior documented |

### 3.2 Incompatible Argo Version

| Test Case ID | TC-NF-002 |
|---|---|
| **Test Case Summary** | Verify behavior with unsupported Argo versions |
| **Test Steps** | 1. Install unsupported Argo version<br/>2. Configure DSPA for external Argo<br/>3. Attempt pipeline execution<br/>4. Document error messages<br/>5. Verify graceful degradation |
| **Expected Results** | - Clear incompatibility errors<br/>- Graceful failure without corruption<br/>- Helpful guidance for resolution |

### 3.3 Missing External Argo

| Test Case ID | TC-NF-003 |
|---|---|
| **Test Case Summary** | Verify behavior when external Argo unavailable |
| **Test Steps** | 1. Configure DSPA for external Argo<br/>2. Stop/remove external Argo service<br/>3. Attempt pipeline submission<br/>4. Restore Argo and verify recovery<br/>5. Check data integrity |
| **Expected Results** | - Clear error messages when Argo unavailable<br/>- Graceful recovery when restored<br/>- No permanent data loss |

### 3.4 Invalid Pipeline Submissions

| Test Case ID | TC-NF-004 |
|---|---|
| **Test Case Summary** | Test invalid pipeline handling with external Argo |
| **Test Steps** | 1. Submit pipelines from `data/pipeline_files/invalid/`<br/>2. Verify appropriate error handling<br/>3. Check error message clarity<br/>4. Ensure no system instability |
| **Expected Results** | - Invalid pipelines rejected appropriately<br/>- Clear error messages provided<br/>- System remains stable<br/>- No resource leaks |

### 3.5 Unsupported Configuration Detection

| Test Case ID | TC-NF-005 |
|---|---|
| **Test Case Summary** | Verify detection of unsupported individual DSPA WorkflowController disable |
| **Test Steps** | 1. Set global WorkflowController management to Removed<br/>2. Attempt to create DSPA with individual `workflowController.deploy: false`<br/>3. Verify appropriate warning/error messages<br/>4. Test documentation guidance for users<br/>5. Ensure configuration is flagged as development-only |
| **Expected Results** | - Unsupported configuration detected<br/>- Clear warning messages displayed<br/>- Documentation provides proper guidance<br/>- Development-only usage clearly indicated |

### 3.6 CRD Version Conflicts

| Test Case ID | TC-NF-006 |
|---|---|
| **Test Case Summary** | Test behavior with conflicting Argo CRD versions |
| **Test Steps** | 1. Install DSP with specific Argo CRD version<br/>2. Install external Argo with different CRD version<br/>3. Attempt pipeline execution<br/>4. Verify conflict detection and resolution<br/>5. Test update-in-place mechanisms |
| **Expected Results** | - CRD version conflicts detected<br/>- Update-in-place works when compatible<br/>- Clear error messages for incompatible versions<br/>- No existing workflow corruption |

### 3.7 Different RBAC Between DSP and External Argo

| Test Case ID | TC-NF-007 |
|---|---|
| **Test Case Summary** | Test DSP and external WorkflowController with different RBAC configurations |
| **Test Steps** | 1. Configure DSP with cluster-level RBAC permissions<br/>2. Install external Argo with namespace-level RBAC restrictions<br/>3. Submit pipelines through DSP interface<br/>4. Verify RBAC conflicts and permission issues<br/>5. Test resource access and execution failures<br/>6. Document RBAC compatibility requirements |
| **Expected Results** | - RBAC conflicts properly identified<br/>- Clear error messages for permission issues<br/>- Guidance provided for RBAC alignment<br/>- No security violations or escalations |

### 3.8 DSP with Incompatible Workflow Schema

| Test Case ID | TC-NF-008 |
|---|---|
| **Test Case Summary** | Test DSP behavior with incompatible workflow schema versions |
| **Test Steps** | 1. Install external Argo with older workflow schema<br/>2. Configure DSP to use external Argo<br/>3. Submit pipelines with newer schema features<br/>4. Verify schema compatibility checking<br/>5. Test graceful degradation or error handling<br/>6. Document schema compatibility matrix |
| **Expected Results** | - Schema incompatibilities detected<br/>- Clear error messages about schema conflicts<br/>- Graceful handling of unsupported features<br/>- No workflow corruption or data loss |

## 4. RBAC and Security Tests

### 4.1 Namespace-Level RBAC

| Test Case ID | TC-RBAC-001 |
|---|---|
| **Test Case Summary** | Verify RBAC with DSP cluster-level and Argo namespace-level access |
| **Test Steps** | 1. Configure DSP with cluster-level permissions<br/>2. Configure Argo with namespace-level restrictions<br/>3. Create users with different permission levels<br/>4. Test pipeline access and execution<br/>5. Verify permission boundaries |
| **Expected Results** | - RBAC properly enforced at both levels<br/>- Users limited to appropriate namespaces<br/>- No unauthorized access to pipelines<br/>- Permission escalation prevented |

### 4.2 Service Account Integration

| Test Case ID | TC-RBAC-002 |
|---|---|
| **Test Case Summary** | Verify service account integration with external Argo |
| **Test Steps** | 1. Configure custom service accounts<br/>2. Set specific RBAC permissions<br/>3. Execute pipelines with different service accounts<br/>4. Verify permission enforcement<br/>5. Test cross-namespace access controls |
| **Expected Results** | - Service accounts properly integrated<br/>- Permissions correctly enforced<br/>- No unauthorized resource access<br/>- Proper audit trail maintained |

### 4.3 Workflow Visibility and Project Access Control

| Test Case ID | TC-RBAC-003 |
|---|---|
| **Test Case Summary** | Verify workflows using external Argo are only visible to users with Project access |
| **Test Steps** | 1. Create multiple Data Science Projects with different users<br/>2. Configure external Argo for all projects<br/>3. Execute pipelines from different projects<br/>4. Test workflow visibility across projects with different users<br/>5. Verify users can only see workflows from their accessible projects<br/>6. Test API access controls and UI filtering<br/>7. Verify external Argo workflows respect DSP project boundaries |
| **Expected Results** | - Workflows only visible to users with project access<br/>- Proper isolation between Data Science Projects<br/>- API and UI enforce access controls correctly<br/>- External Argo workflows respect DSP boundaries<br/>- No cross-project workflow visibility |

## 5. Boundary Tests

### 5.1 Resource Limits

| Test Case ID | TC-BT-001 |
|---|---|
| **Test Case Summary** | Verify behavior at resource boundaries |
| **Test Steps** | 1. Configure external Argo with resource limits<br/>2. Submit resource-intensive pipelines<br/>3. Monitor resource utilization<br/>4. Verify appropriate throttling<br/>5. Test recovery when resources available |
| **Expected Results** | - Resource limits properly enforced<br/>- Appropriate queuing/throttling behavior<br/>- Clear resource constraint messages<br/>- Graceful recovery when resources free |

### 5.2 Large Artifact Handling

| Test Case ID | TC-BT-002 |
|---|---|
| **Test Case Summary** | Verify handling of large pipeline artifacts |
| **Test Steps** | 1. Configure pipelines with large data artifacts<br/>2. Execute with external Argo<br/>3. Monitor storage and transfer performance<br/>4. Verify artifact integrity<br/>5. Test cleanup mechanisms |
| **Expected Results** | - Large artifacts handled efficiently<br/>- No data corruption or loss<br/>- Acceptable transfer performance<br/>- Proper cleanup after completion |

### 5.3 High Concurrency

| Test Case ID | TC-BT-003 |
|---|---|
| **Test Case Summary** | Test high concurrency scenarios |
| **Test Steps** | 1. Submit multiple concurrent pipelines<br/>2. Monitor external Argo performance<br/>3. Verify all pipelines complete<br/>4. Check for resource contention<br/>5. Validate result consistency |
| **Expected Results** | - High concurrency handled appropriately<br/>- No pipeline failures due to contention<br/>- Consistent execution results<br/>- Stable system performance |

## 6. Performance Tests

### 6.1 Execution Performance Comparison

| Test Case ID | TC-PT-001 |
|---|---|
| **Test Case Summary** | Compare performance between internal and external Argo |
| **Test Steps** | 1. Execute identical pipeline suite with internal WC<br/>2. Execute same suite with external Argo<br/>3. Measure execution times and resource usage<br/>4. Compare throughput and latency<br/>5. Document performance characteristics |
| **Expected Results** | - Performance with external Argo acceptable<br/>- No significant degradation vs internal WC<br/>- Resource utilization within bounds<br/>- Scalability maintained |

### 6.2 Startup and Initialization

| Test Case ID | TC-PT-002 |
|---|---|
| **Test Case Summary** | Measure DSPA startup time with external Argo |
| **Test Steps** | 1. Measure DSPA creation time with internal WC<br/>2. Measure DSPA creation time with external Argo<br/>3. Compare initialization times<br/>4. Monitor resource usage during startup<br/>5. Document timing differences |
| **Expected Results** | - Startup time with external Argo reasonable<br/>- Initialization completes successfully<br/>- Resource usage during startup acceptable<br/>- No significant delays |

## 7. Compatibility Matrix Tests

### 7.1 Current Version (N) Compatibility

| Test Case ID | TC-CM-001 |
|---|---|
| **Test Case Summary** | Validate compatibility with current supported Argo version |
| **Test Steps** | 1. Install current supported Argo version (e.g., 3.4.16)<br/>2. Configure DSPA for external Argo<br/>3. Execute comprehensive pipeline test suite<br/>4. Verify all features work correctly<br/>5. Document any limitations |
| **Expected Results** | - Full compatibility with current version<br/>- All pipeline features operational<br/>- No breaking changes or issues<br/>- Performance within acceptable range |

### 7.2 Previous Version (N-1) Compatibility

| Test Case ID | TC-CM-002 |
|---|---|
| **Test Case Summary** | Validate compatibility with previous supported Argo version |
| **Test Steps** | 1. Install previous supported Argo version (e.g., 3.4.15)<br/>2. Configure DSPA for external Argo<br/>3. Execute comprehensive pipeline test suite<br/>4. Document compatibility differences<br/>5. Verify core functionality maintained |
| **Expected Results** | - Core functionality works with N-1 version<br/>- Any limitations clearly documented<br/>- No critical failures or data loss<br/>- Upgrade path available |

### 7.2.1 Z-Stream Version Compatibility

| Test Case ID | TC-CM-002a |
|---|---|
| **Test Case Summary** | Validate compatibility with z-stream (patch) versions of Argo |
| **Test Steps** | 1. Test current DSP with multiple z-stream versions of same minor Argo release<br/>2. Example: Test DSP v3.4.17 with Argo v3.4.16, v3.4.17, v3.4.18<br/>3. Execute standard pipeline test suite for each z-stream version<br/>4. Document any breaking changes in patch versions<br/>5. Verify backward and forward compatibility within minor version |
| **Expected Results** | - Z-stream versions maintain compatibility<br/>- No breaking changes in patch releases<br/>- Smooth operation across patch versions<br/>- Clear documentation of any exceptions |

### 7.3 Version Matrix Validation

| Test Case ID | TC-CM-003 |
|---|---|
| **Test Case Summary** | Systematically validate compatibility matrix |
| **Test Steps** | 1. For each version in compatibility matrix:<br/>   a. Deploy specific Argo version<br/>   b. Configure DSPA<br/>   c. Execute standard test suite<br/>   d. Document results and issues<br/>2. Update compatibility matrix<br/>3. Identify unsupported combinations |
| **Expected Results** | - Compatibility matrix accurately reflects reality<br/>- All supported versions documented<br/>- Unsupported combinations identified<br/>- Clear guidance for version selection |

### 7.4 DSP and External Argo Co-existence Validation

| Test Case ID | TC-CM-004 |
|---|---|
| **Test Case Summary** | Validate successful hello world pipeline with DSP and External Argo co-existing |
| **Test Steps** | 1. Deploy DSPA with internal WorkflowController<br/>2. Install external Argo WorkflowController on same cluster<br/>3. Submit simple hello world pipeline through DSP<br/>4. Verify pipeline executes successfully using DSP controller<br/>5. Verify external Argo remains unaffected<br/>6. Test pipeline monitoring and status reporting<br/>7. Validate artifact handling and logs access |
| **Expected Results** | - Hello world pipeline executes successfully<br/>- DSP WorkflowController processes the pipeline<br/>- External Argo WorkflowController unaffected<br/>- No resource conflicts or interference<br/>- Pipeline status and logs accessible<br/>- Artifacts properly stored and retrievable |

### 7.5 API Server and WorkflowController Compatibility

| Test Case ID | TC-CM-005 |
|---|---|
| **Test Case Summary** | Verify DSP API Server compatibility with different external WorkflowController versions |
| **Test Steps** | 1. Deploy DSP API Server with specific Argo library dependencies<br/>2. Install external Argo WorkflowController with different version<br/>3. Test API Server to WorkflowController communication<br/>4. Verify Kubernetes API interactions (CRs, status updates)<br/>5. Test pipeline submission, execution, and status reporting<br/>6. Monitor for API compatibility issues or version mismatches<br/>7. Document API compatibility matrix |
| **Expected Results** | - API Server communicates successfully with external WC<br/>- Kubernetes API interactions work correctly<br/>- Pipeline lifecycle management functions properly<br/>- Status updates and monitoring work correctly<br/>- API compatibility documented and validated |

## 8. Uninstall and Data Preservation Tests

### 8.1 DSPA Uninstall with External Argo

| Test Case ID | TC-UP-001 |
|---|---|
| **Test Case Summary** | Verify DSPA uninstall behavior with external Argo |
| **Test Steps** | 1. Configure DSPA with external Argo (no internal WC)<br/>2. Execute multiple pipelines and generate data<br/>3. Delete DSPA<br/>4. Verify external Argo WorkflowController remains intact<br/>5. Verify DSPA-specific resources are cleaned up<br/>6. Check that pipeline history is appropriately handled |
| **Expected Results** | - DSPA removes cleanly<br/>- External Argo WorkflowController unaffected<br/>- No impact on other DSPAs using same external Argo<br/>- Pipeline data handling follows standard procedures |

### 8.2 DSPA Uninstall with Internal WorkflowController

| Test Case ID | TC-UP-002 |
|---|---|
| **Test Case Summary** | Verify standard DSPA uninstall with internal WorkflowController |
| **Test Steps** | 1. Configure DSPA with internal WorkflowController<br/>2. Execute pipelines and generate data<br/>3. Delete DSPA<br/>4. Verify WorkflowController is removed with DSPA<br/>5. Verify proper cleanup of all DSPA components<br/>6. Ensure no external Argo impact |
| **Expected Results** | - DSPA and WorkflowController removed completely<br/>- Standard cleanup procedures followed<br/>- No resource leaks or orphaned components<br/>- External Argo installations unaffected |

### 8.3 Data Preservation During WorkflowController Transitions

| Test Case ID | TC-UP-003 |
|---|---|
| **Test Case Summary** | Verify data preservation during WorkflowController management transitions |
| **Test Steps** | 1. Create DSPA with internal WC and execute pipelines<br/>2. Disable WC globally (transition to external Argo)<br/>3. Verify run history, artifacts, and metadata preserved<br/>4. Re-enable WC globally (transition back to internal)<br/>5. Verify all historical data remains accessible<br/>6. Test new pipeline execution in both states |
| **Expected Results** | - Pipeline run history preserved across transitions<br/>- Artifacts remain accessible<br/>- Metadata integrity maintained<br/>- New pipelines work in both configurations |

### 8.4 WorkflowTemplates and CronWorkflows Preservation

| Test Case ID | TC-UP-004 |
|---|---|
| **Test Case Summary** | Verify preservation of WorkflowTemplates and CronWorkflows during DSP install/uninstall |
| **Test Steps** | 1. Install external Argo and create WorkflowTemplates and CronWorkflows<br/>2. Install DSP with BYOAW configuration<br/>3. Verify existing WorkflowTemplates and CronWorkflows remain intact<br/>4. Create additional WorkflowTemplates through DSP interface<br/>5. Uninstall DSP components<br/>6. Verify all WorkflowTemplates and CronWorkflows still exist<br/>7. Test functionality of preserved resources with external Argo |
| **Expected Results** | - Pre-existing WorkflowTemplates and CronWorkflows preserved<br/>- DSP-created templates also preserved during uninstall<br/>- All preserved resources remain functional<br/>- No data corruption or resource deletion<br/>- External Argo can use all preserved templates |

## 9. Migration and Upgrade Tests

### 9.1 DSP-Managed to External Migration

| Test Case ID | TC-MU-001 |
|---|---|
| **Test Case Summary** | Verify migration from DSP-managed to external Argo |
| **Test Steps** | 1. Create DSPA with internal WorkflowController<br/>2. Execute pipelines and accumulate data<br/>3. Install external Argo<br/>4. Disable internal WCs globally<br/>5. Verify data preservation and new execution |
| **Expected Results** | - Migration completes without data loss<br/>- Historical data remains accessible<br/>- New pipelines use external Argo<br/>- Artifacts and metadata preserved |

### 9.2 External to DSP-Managed Migration

| Test Case ID | TC-MU-002 |
|---|---|
| **Test Case Summary** | Verify migration from external to DSP-managed Argo |
| **Test Steps** | 1. Configure DSPA with external Argo<br/>2. Execute pipelines and verify data<br/>3. Re-enable internal WCs globally<br/>4. Remove external Argo configuration<br/>5. Verify continued operation |
| **Expected Results** | - Migration to internal WC successful<br/>- Pipeline history preserved<br/>- New pipelines use internal WC<br/>- No service interruption |

### 9.3 RHOAI Upgrade Scenarios

| Test Case ID | TC-MU-003 |
|---|---|
| **Test Case Summary** | Verify RHOAI upgrade preserves external Argo setup |
| **Test Steps** | 1. Configure RHOAI with external Argo<br/>2. Execute baseline pipeline tests<br/>3. Upgrade RHOAI to newer version<br/>4. Verify external Argo configuration intact<br/>5. Re-execute pipeline tests |
| **Expected Results** | - Upgrade preserves BYOAW configuration<br/>- External Argo continues working<br/>- No functionality regression<br/>- Configuration settings maintained |

### 9.4 Argo Version Upgrade with External Installation

| Test Case ID | TC-MU-004 |
|---|---|
| **Test Case Summary** | Verify external Argo version upgrade scenarios |
| **Test Steps** | 1. Configure DSPA with external Argo version N-1<br/>2. Execute baseline pipeline tests<br/>3. Upgrade external Argo to version N<br/>4. Verify compatibility matrix adherence<br/>5. Test pipeline execution post-upgrade<br/>6. Document any required RHOAI updates |
| **Expected Results** | - External Argo upgrade completes successfully<br/>- Compatibility maintained within support matrix<br/>- Clear guidance for required RHOAI updates<br/>- Pipeline functionality preserved |

### 9.5 Independent Lifecycle Management

| Test Case ID | TC-MU-005 |
|---|---|
| **Test Case Summary** | Verify independent lifecycle management of RHOAI and external Argo |
| **Test Steps** | 1. Install and configure RHOAI with external Argo<br/>2. Perform independent upgrade of external Argo installation<br/>3. Verify RHOAI continues operating without issues<br/>4. Perform independent upgrade of RHOAI<br/>5. Verify external Argo continues operating without issues<br/>6. Test independent scaling of each component<br/>7. Verify independent maintenance and restart scenarios |
| **Expected Results** | - Independent upgrades work without mutual interference<br/>- Each component maintains functionality during the other's maintenance<br/>- Scaling operations work independently<br/>- No forced coupling of upgrade/maintenance schedules<br/>- Clear documentation of independence boundaries |

## Test Execution Schedule

### Phase 1: Foundation (Weeks 1-3)
- Cluster Configuration Tests (TC-CC-001 to TC-CC-009)
- Basic Positive Functional Tests (TC-PF-001, TC-PF-002)
- Basic Negative Tests (TC-NF-001, TC-NF-002)
- Pre-existing Argo Detection and CRD Testing (TC-CC-008, TC-CC-009)

### Phase 2: Compatibility and Integration (Weeks 4-5)
- Compatibility Matrix Tests (TC-CM-001 to TC-CM-005)
- Z-Stream Version Testing (TC-CM-002a)
- RBAC and Security Tests (TC-RBAC-001 to TC-RBAC-003)
- Advanced Positive Tests (TC-PF-003, TC-PF-004)

### Phase 3: Conflict Resolution and Negative Testing (Weeks 6-7)
- Extended Negative Tests (TC-NF-003 to TC-NF-008)
- Co-existence Testing (TC-NF-001a)
- API Server Compatibility Testing (TC-CM-005)

### Phase 4: Advanced Scenarios (Weeks 8-9)
- Uninstall and Data Preservation Tests (TC-UP-001 to TC-UP-004)
- Migration and Upgrade Tests (TC-MU-001 to TC-MU-005)
- Performance Tests (TC-PT-001, TC-PT-002)
- Boundary Tests (TC-BT-001 to TC-BT-003)

## Success Criteria

### Must Have
- All positive functional tests pass without failures
- Compatibility matrix validation complete for N and N-1 versions
- Z-stream (patch) version compatibility validated
- Migration scenarios preserve data integrity
- Security and RBAC properly enforced
- Performance within acceptable bounds (no >20% degradation)
- Platform-level CRD and RBAC management works correctly
- Data preservation during WorkflowController transitions
- Sub-component removal functionality validated
- Pre-existing Argo detection and prevention working
- CRD update-in-place functionality validated
- WorkflowTemplates and CronWorkflows preservation confirmed
- API Server to WorkflowController compatibility verified
- Workflow visibility and project access controls enforced

### Should Have
- Negative test scenarios handled gracefully
- Clear error messages for all failure modes
- Unsupported configuration detection functional
- CRD version conflict resolution working
- RBAC conflict detection and resolution
- Schema compatibility validation working
- Co-existence scenarios validated successfully
- Independent lifecycle management validated
- Documentation complete and accurate
- Uninstall scenarios preserve external Argo integrity

### Could Have
- Performance optimizations for external Argo scenarios
- Enhanced monitoring and observability
- Additional version compatibility beyond N-1
- Automated detection of conflicting configurations
- Advanced CRD update-in-place mechanisms

## Risk Assessment

### High Risk
- Data loss during migration scenarios
- Security vulnerabilities in multi-tenant setups
- Performance degradation with external Argo
- Incompatibility with future Argo versions

### Medium Risk
- Complex configuration management
- Upgrade complications
- Resource contention in shared scenarios
- Error handling gaps

### Low Risk
- Minor UI/UX inconsistencies
- Documentation completeness
- Non-critical performance variations
- Edge case handling

## Test Deliverables

1. **Test Execution Reports** - Detailed results for each test phase with comprehensive coverage
2. **Enhanced Compatibility Matrix** - Validated version combinations including Z-stream compatibility and API compatibility
3. **Performance Benchmarks** - Comparative analysis of internal vs external Argo across all scenarios
4. **Comprehensive Security Assessment** - RBAC and isolation validation including project access controls
5. **Migration Documentation** - Complete procedures for all migration scenarios and lifecycle management
6. **Data Preservation Guidelines** - Best practices for maintaining data integrity during all transitions
7. **Uninstall Procedures** - Validated procedures for clean removal preserving WorkflowTemplates and CronWorkflows
8. **CRD Management Guidelines** - Platform-level CRD update-in-place and conflict resolution procedures
9. **Pre-existing Argo Detection Guide** - Implementation and configuration of detection mechanisms
10. **Configuration Validation Guide** - Detection and resolution of all unsupported configurations
11. **RBAC Compatibility Matrix** - Comprehensive guidelines for DSP and external Argo RBAC alignment
12. **Schema Compatibility Guide** - Workflow schema version compatibility and API compatibility matrix
13. **Co-existence Best Practices** - Detailed recommendations for running DSP and external Argo together
14. **Z-Stream Testing Strategy** - Framework for ongoing patch version compatibility validation
15. **API Compatibility Documentation** - DSP API Server to external WorkflowController compatibility guidelines
16. **Independent Lifecycle Management Guide** - Best practices for managing RHOAI and Argo independently
17. **Known Issues Log** - Comprehensive documentation of limitations and workarounds
18. **Final Test Report** - Executive summary with recommendations, lessons learned, and future testing strategy
