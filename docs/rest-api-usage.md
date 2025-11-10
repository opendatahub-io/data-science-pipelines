# Using the Data Science Pipelines REST API with Kubeflow Pipelines SDK

This document provides a comprehensive guide on how to interact with the Data Science Pipelines REST API using the Kubeflow Pipelines SDK. The Data Science Pipelines service provides a RESTful API that allows you to programmatically manage pipelines, experiments, runs, and other resources.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Client Setup](#client-setup)
- [Core Operations](#core-operations)
  - [Working with Pipelines](#working-with-pipelines)
  - [Working with Pipeline Versions](#working-with-pipeline-versions)
  - [Working with Experiments](#working-with-experiments)
  - [Working with Runs](#working-with-runs)
  - [Working with Recurring Runs](#working-with-recurring-runs)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [Additional Resources](#additional-resources)

## Overview

The Data Science Pipelines REST API provides endpoints for:
- Managing pipelines and pipeline versions
- Creating and managing experiments
- Submitting and monitoring pipeline runs
- Managing recurring runs (scheduled executions)
- Retrieving artifacts and metrics

The Kubeflow Pipelines SDK provides Python client libraries that wrap these REST API calls, making it easier to interact with the service programmatically.

## Prerequisites

Before using the REST API, ensure you have:

1. **Kubeflow Pipelines SDK installed**:
   ```bash
   pip install kfp
   ```

2. **Access to a Data Science Pipelines deployment** with the API server endpoint URL

3. **Appropriate authentication credentials** (if authentication is enabled)

## Client Setup

### Retrieving the DSP API URL on OpenShift

When deploying Data Science Pipelines on OpenShift, the API server is exposed via an OpenShift Route. To retrieve the API URL, you can use the `oc` CLI tool:

```bash
# Set your namespace and DSP Custom Resource name
DSP_NAMESPACE="your-namespace"
DSPA_NAME="your-dspa-name"

# Retrieve the route hostname
DSP_ROUTE=$(oc get routes -n ${DSP_NAMESPACE} ds-pipeline-${DSPA_NAME} --template={{.spec.host}})

# The API URL will be https://${DSP_ROUTE}
echo "DSP API URL: https://${DSP_ROUTE}"
```

The route name follows the pattern `ds-pipeline-{DSPA_NAME}`, where `{DSPA_NAME}` is the `metadata.name` of your `DataSciencePipelinesApplication` Custom Resource.

You can then use this URL when initializing the client:

```python
import os
import kfp
from kfp import Client

# Get the route from environment variable or retrieve it programmatically
dsp_api_url = os.getenv('DSP_ROUTE', 'https://ds-pipeline-<dspa_name>.some.openshift.host.com')
client = Client(host=dsp_api_url)
```

Alternatively, you can retrieve the route programmatically using the OpenShift Python client:

```python
from kubernetes import client, config
from openshift.dynamic import DynamicClient

# Load kubeconfig
k8s_client = config.new_client_from_config()
dyn_client = DynamicClient(k8s_client)

# Get the route
v1_routes = dyn_client.resources.get(api_version='route.openshift.io/v1', kind='Route')
route = v1_routes.get(name=f'ds-pipeline-{dsp_cr_name}', namespace=dsp_namespace)

# Extract the hostname
api_url = f"https://{route.spec.host}"
client = Client(host=api_url)
```

### Basic Client Configuration

The simplest way to create a client is by providing the API server host URL.

```python
import kfp
from kfp import Client

# For remote access with HTTPS
client = Client(host='https://ds-pipeline-<dspa_name>.some.openshift.host.com')

# With custom namespace (for multi-tenant deployments)
client = Client(
    host='https://ds-pipeline-<dspa_name>.some.openshift.host.com',
    namespace='your-namespace'
)
```

### Client with Custom Configuration

You can customize the client behavior by specifying additional parameters such as timeout and retry settings.

```python
import kfp
from kfp import Client

# With custom timeout and retry settings
client = Client(
    host='https://ds-pipeline-<dspa_name>.some.openshift.host.com',
    namespace='your-namespace',  # Optional: specify namespace for multi-tenant deployments
    timeout=300,  # Optional: request timeout in seconds (default: 300)
    retry_count=3,  # Optional: number of retries for failed requests (default: 3)
    verify_ssl=True,  # Optional: verify SSL certificates (default: True)
    ssl_ca_cert='/path/to/ca-cert.pem'  # Optional: path to CA certificate file
)
```

## Core Operations

This section covers the core operations for working with Data Science Pipelines, including managing pipelines, experiments, runs, and recurring runs.

### Working with Pipelines

Pipelines are the core building blocks of Data Science Pipelines. This section covers how to upload, list, retrieve, and delete pipelines using the REST API client.

### Uploading a Pipeline

Upload pipelines from local files or remote URLs to make them available for execution.

```python
# Upload from a local file
pipeline = client.upload_pipeline(
    pipeline_package_path='path/to/pipeline.yaml',
    pipeline_name='my-pipeline',
    description='My sample pipeline'
)

# Upload from a URL
pipeline = client.upload_pipeline_from_url(
    pipeline_url='https://github.com/example/pipeline.yaml',
    pipeline_name='remote-pipeline'
)
```

### Listing Pipelines

Retrieve a list of pipelines with support for pagination and filtering.

```python
# List all pipelines
pipelines = client.list_pipelines()
print(f"Found {pipelines.total_size} pipelines")

for pipeline in pipelines.pipelines:
    print(f"Pipeline: {pipeline.display_name} (ID: {pipeline.pipeline_id})")

# List with pagination
pipelines = client.list_pipelines(page_size=10, page_token=None)

# List with filtering
pipelines = client.list_pipelines(filter='name="my-pipeline"')
```

### Getting Pipeline Details

Retrieve detailed information about a specific pipeline by ID or name.

```python
# Get pipeline by ID
pipeline_id = "your-pipeline-id"
pipeline = client.get_pipeline(pipeline_id)

# Get pipeline by name
pipeline = client.get_pipeline_by_name('my-pipeline')
```

### Deleting a Pipeline

Remove pipelines from the system by ID or by first retrieving the pipeline by name.

```python
# Delete pipeline by ID
client.delete_pipeline(pipeline_id)

# Delete pipeline by name
pipeline = client.get_pipeline_by_name('my-pipeline')
client.delete_pipeline(pipeline.pipeline_id)
```

### Working with Pipeline Versions

Pipeline versions allow you to maintain multiple versions of the same pipeline, enabling version control and tracking of pipeline changes over time.

### Creating Pipeline Versions

Create new versions of existing pipelines by uploading updated pipeline definitions from local files or remote URLs.

```python
# Create a new version of an existing pipeline
version = client.upload_pipeline_version(
    pipeline_package_path='path/to/updated-pipeline.yaml',
    pipeline_version_name='v2.0',
    pipeline_id=pipeline_id
)

# Create version from URL
version = client.upload_pipeline_version_from_url(
    pipeline_url='https://github.com/example/pipeline-v2.yaml',
    pipeline_version_name='v2.1',
    pipeline_id=pipeline_id
)
```

### Listing Pipeline Versions

Retrieve all versions associated with a specific pipeline.

```python
# List all versions of a pipeline
versions = client.list_pipeline_versions(pipeline_id)

for version in versions.pipeline_versions:
    print(f"Version: {version.display_name} (ID: {version.pipeline_version_id})")
```

### Getting Pipeline Version Details

Retrieve detailed information about a specific pipeline version.

```python
# Get specific version
version = client.get_pipeline_version(
    pipeline_id=pipeline_id,
    version_id=version_id
)
```

### Working with Experiments

Experiments provide a way to organize and group related pipeline runs, making it easier to track and compare results across multiple executions.

### Creating Experiments

Create new experiments to organize your pipeline runs into logical groups.

```python
# Create a new experiment
experiment = client.create_experiment(
    name='my-experiment',
    description='Experiment for testing pipelines',
    namespace='default'  # Optional, uses client's default namespace
)
```

### Listing Experiments

Retrieve a list of experiments with support for pagination and sorting.

```python
# List all experiments
experiments = client.list_experiments()

for exp in experiments.experiments:
    print(f"Experiment: {exp.display_name} (ID: {exp.experiment_id})")

# List with pagination and sorting
experiments = client.list_experiments(
    page_size=20,
    sort_by='created_at desc'
)
```

### Getting Experiment Details

Retrieve detailed information about a specific experiment by ID or name.

```python
# Get experiment by ID
experiment = client.get_experiment(experiment_id)

# Get experiment by name
experiment = client.get_experiment_by_name('my-experiment')
```

### Working with Runs

Runs represent individual executions of pipelines. This section covers how to submit, monitor, list, and manage pipeline runs.

### Creating and Submitting Runs

Submit pipeline runs with parameters, either from uploaded pipelines or directly from pipeline package files.

```python
# Submit a run with parameters
run = client.run_pipeline(
    experiment_id=experiment.experiment_id,
    job_name='my-pipeline-run',
    pipeline_id=pipeline_id,
    params={
        'input_path': 's3://my-bucket/input.csv',
        'output_path': 's3://my-bucket/output/',
        'learning_rate': 0.01
    }
)

# Submit run with specific pipeline version
run = client.run_pipeline(
    experiment_id=experiment.experiment_id,
    job_name='my-pipeline-run-v2',
    pipeline_id=pipeline_id,
    version_id=version_id,
    params={'param1': 'value1'}
)

# Submit run from pipeline package
run = client.create_run_from_pipeline_package(
    pipeline_file='path/to/pipeline.yaml',
    arguments={'param1': 'value1'},
    run_name='direct-run',
    experiment_name='my-experiment'
)
```

### Monitoring Runs

Track the progress and status of pipeline runs, including waiting for completion.

```python
# Get run details
run_detail = client.get_run(run.run_id)
print(f"Run status: {run_detail.run.status}")

# Wait for run completion
client.wait_for_run_completion(run.run_id, timeout=3600)  # 1 hour timeout

# Get run status
status = client.get_run_status(run.run_id)
print(f"Current status: {status}")
```

### Listing Runs

Retrieve a list of runs with support for filtering, sorting, and pagination.

```python
# List all runs
runs = client.list_runs()

# List runs for a specific experiment
runs = client.list_runs(experiment_id=experiment.experiment_id)

# List runs with filtering
runs = client.list_runs(
    filter='status="Running"',
    sort_by='created_at desc',
    page_size=50
)

for run in runs.runs:
    print(f"Run: {run.display_name} - Status: {run.status}")
```

### Managing Run Lifecycle

Control the lifecycle of runs by canceling, deleting, archiving, or unarchiving them.

```python
# Cancel a running pipeline
client.cancel_run(run.run_id)

# Delete a run
client.delete_run(run.run_id)

# Archive a run
client.archive_run(run.run_id)

# Unarchive a run
client.unarchive_run(run.run_id)
```

### Working with Recurring Runs

Recurring runs enable you to schedule pipelines to run automatically at specified intervals, making it easy to automate repetitive tasks.

### Creating Recurring Runs (Scheduled Pipelines)

Create scheduled pipeline executions that run automatically based on time-based triggers.

```python
from kfp.client.recurring_run import PeriodicSchedule
import datetime

# Create a daily recurring run
schedule = PeriodicSchedule(
    start_time=datetime.datetime.now(),
    end_time=datetime.datetime.now() + datetime.timedelta(days=30),
    interval_second=24*60*60  # Daily
)

recurring_run = client.create_recurring_run(
    experiment_id=experiment.experiment_id,
    job_name='daily-pipeline',
    pipeline_id=pipeline_id,
    params={'date': '{{workflow.creationTimestamp}}'},
    trigger=schedule,
    max_concurrency=1
)
```

### Managing Recurring Runs

List, retrieve, enable, disable, and delete recurring runs.

```python
# List recurring runs
recurring_runs = client.list_recurring_runs()

# Get recurring run details
recurring_run = client.get_recurring_run(recurring_run_id)

# Enable/disable recurring run
client.enable_recurring_run(recurring_run_id)
client.disable_recurring_run(recurring_run_id)

# Delete recurring run
client.delete_recurring_run(recurring_run_id)
```

## Error Handling

### Common Error Patterns

Handle common API errors by catching ApiException and checking status codes to provide appropriate error handling and logging.

```python
from kfp.client.exceptions import ApiException
import logging

def safe_pipeline_operation():
    try:
        # Your pipeline operation
        run = client.run_pipeline(
            experiment_id=experiment_id,
            job_name='test-run',
            pipeline_id=pipeline_id
        )
        return run
    
    except ApiException as e:
        if e.status == 404:
            logging.error("Pipeline or experiment not found")
        elif e.status == 401:
            logging.error("Authentication failed")
        elif e.status == 403:
            logging.error("Permission denied")
        else:
            logging.error(f"API error: {e.status} - {e.reason}")
        raise
    
    except Exception as e:
        logging.error(f"Unexpected error: {str(e)}")
        raise
```

### Retry Logic

Implement retry mechanisms with exponential backoff to handle transient failures and improve reliability of API calls.

```python
import time
from functools import wraps

def retry_on_failure(max_retries=3, delay=1):
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            for attempt in range(max_retries):
                try:
                    return func(*args, **kwargs)
                except ApiException as e:
                    if e.status >= 500 and attempt < max_retries - 1:
                        time.sleep(delay * (2 ** attempt))  # Exponential backoff
                        continue
                    raise
            return None
        return wrapper
    return decorator

@retry_on_failure(max_retries=3)
def submit_pipeline_with_retry():
    return client.run_pipeline(
        experiment_id=experiment_id,
        job_name='retry-run',
        pipeline_id=pipeline_id
    )
```

## Best Practices

### 1. Resource Management

Use try-finally blocks or context managers to ensure proper cleanup of resources, even when errors occur.

```python
# Always use try-finally or context managers for cleanup
try:
    run = client.run_pipeline(...)
    # Monitor run
    client.wait_for_run_completion(run.run_id)
finally:
    # Cleanup if needed
    pass
```

### 2. Efficient Pagination

Implement pagination to efficiently retrieve large datasets by iterating through pages using page tokens.

```python
def get_all_pipelines(client):
    """Efficiently retrieve all pipelines using pagination."""
    all_pipelines = []
    page_token = None
    
    while True:
        response = client.list_pipelines(
            page_size=100,  # Use reasonable page size
            page_token=page_token
        )
        
        all_pipelines.extend(response.pipelines)
        
        if not response.next_page_token:
            break
            
        page_token = response.next_page_token
    
    return all_pipelines
```

### 3. Parameter Validation

Validate parameters before submitting pipeline runs to catch errors early and provide better error messages.

```python
def validate_and_submit_run(client, pipeline_id, experiment_id, params):
    """Validate parameters before submitting a run."""
    
    # Get pipeline to check required parameters
    pipeline = client.get_pipeline(pipeline_id)
    
    # Validate required parameters exist
    # (Implementation depends on your pipeline structure)
    
    return client.run_pipeline(
        experiment_id=experiment_id,
        job_name=f'validated-run-{int(time.time())}',
        pipeline_id=pipeline_id,
        params=params
    )
```

### 4. Logging and Monitoring

Implement comprehensive logging and monitoring to track pipeline execution progress and diagnose issues.

```python
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def monitored_pipeline_run(client, **kwargs):
    """Submit and monitor a pipeline run with logging."""
    
    logger.info(f"Submitting pipeline run with params: {kwargs.get('params', {})}")
    
    run = client.run_pipeline(**kwargs)
    logger.info(f"Run submitted with ID: {run.run_id}")
    
    # Monitor progress
    while True:
        run_detail = client.get_run(run.run_id)
        status = run_detail.run.status
        
        logger.info(f"Run {run.run_id} status: {status}")
        
        if status in ['Succeeded', 'Failed', 'Cancelled']:
            break
            
        time.sleep(30)  # Check every 30 seconds
    
    return run_detail
```

## Examples

### Complete Workflow Example

A complete end-to-end example demonstrating how to create an experiment, upload a pipeline, submit a run, and monitor its execution.

```python
#!/usr/bin/env python3
"""
Complete example of using Data Science Pipelines REST API
"""

import kfp
from kfp import Client
import time
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def main():
    # Initialize client
    client = Client(host='http://localhost:8080')
    
    try:
        # 1. Create or get experiment
        experiment_name = 'rest-api-example'
        try:
            experiment = client.get_experiment_by_name(experiment_name)
            logger.info(f"Using existing experiment: {experiment.experiment_id}")
        except:
            experiment = client.create_experiment(
                name=experiment_name,
                description='Example experiment using REST API'
            )
            logger.info(f"Created new experiment: {experiment.experiment_id}")
        
        # 2. Upload pipeline
        pipeline = client.upload_pipeline(
            pipeline_package_path='path/to/your/pipeline.yaml',
            pipeline_name='example-pipeline',
            description='Example pipeline for REST API demo'
        )
        logger.info(f"Uploaded pipeline: {pipeline.pipeline_id}")
        
        # 3. Submit run
        run = client.run_pipeline(
            experiment_id=experiment.experiment_id,
            job_name=f'example-run-{int(time.time())}',
            pipeline_id=pipeline.pipeline_id,
            params={
                'input_data': 'gs://your-bucket/data.csv',
                'model_name': 'example-model',
                'epochs': 10
            }
        )
        logger.info(f"Submitted run: {run.run_id}")
        
        # 4. Monitor run
        logger.info("Monitoring run progress...")
        client.wait_for_run_completion(run.run_id, timeout=3600)
        
        # 5. Get final results
        run_detail = client.get_run(run.run_id)
        final_status = run_detail.run.status
        logger.info(f"Run completed with status: {final_status}")
        
        if final_status == 'Succeeded':
            logger.info("Pipeline executed successfully!")
        else:
            logger.error(f"Pipeline failed with status: {final_status}")
            
    except Exception as e:
        logger.error(f"Error in pipeline execution: {str(e)}")
        raise

if __name__ == '__main__':
    main()
```

### Batch Processing Example

Process multiple pipelines in batch with comprehensive error handling to manage large-scale pipeline submissions.

```python
def batch_process_pipelines(client, pipeline_configs):
    """
    Process multiple pipelines in batch with error handling.
    
    Args:
        client: KFP client instance
        pipeline_configs: List of pipeline configuration dictionaries
    """
    results = []
    
    for config in pipeline_configs:
        try:
            # Submit run
            run = client.run_pipeline(
                experiment_id=config['experiment_id'],
                job_name=config['job_name'],
                pipeline_id=config['pipeline_id'],
                params=config.get('params', {})
            )
            
            results.append({
                'config': config,
                'run_id': run.run_id,
                'status': 'submitted',
                'error': None
            })
            
        except Exception as e:
            results.append({
                'config': config,
                'run_id': None,
                'status': 'failed',
                'error': str(e)
            })
    
    return results

# Usage
configs = [
    {
        'experiment_id': 'exp-1',
        'job_name': 'batch-job-1',
        'pipeline_id': 'pipeline-1',
        'params': {'input': 'data1.csv'}
    },
    {
        'experiment_id': 'exp-1',
        'job_name': 'batch-job-2',
        'pipeline_id': 'pipeline-1',
        'params': {'input': 'data2.csv'}
    }
]

results = batch_process_pipelines(client, configs)
```

## Additional Resources

### SDK Documentation

The Kubeflow Pipelines SDK provides Python client libraries that simplify interaction with the Data Science Pipelines REST API. For more detailed information about using the Kubeflow Pipelines SDK, refer to the official SDK documentation:

- [Kubeflow Pipelines SDK documentation](https://kubeflow-pipelines.readthedocs.io/)

This documentation provides comprehensive information about the SDK's client libraries, methods, and usage patterns to help you effectively interact with the Data Science Pipelines REST API.

### Full REST API Documentation

For comprehensive details on the Kubeflow Pipelines REST API, including complete endpoint specifications, request/response schemas, and additional usage examples, refer to the official API reference:

- [Kubeflow Pipelines API Reference](https://www.kubeflow.org/docs/components/pipelines/reference/api/kubeflow-pipeline-api-spec/)

This documentation provides in-depth information about all available REST API endpoints and can help you explore advanced features and capabilities beyond what is covered in this guide.





