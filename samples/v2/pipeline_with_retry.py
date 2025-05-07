import os

from kfp import dsl

from kfp import compiler

_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')

@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def verify_retries(retries: str) -> bool:
    if retries != '2':
        raise Exception('Number of retries has not reached two yet.')
    return True

@dsl.component(kfp_package_path=_KFP_PACKAGE_PATH)
def print_op(text: str) -> str:
    print(text)
    return text

@dsl.pipeline
def nested_pipeline():
    subtask1 = verify_retries(retries='{{retries}}').set_retry(num_retries=2)
    subtask2 = print_op(text='test').set_retry(num_retries=0)
    subtask3 = print_op(text='test').set_retry(num_retries=2, backoff_duration="1s", backoff_factor=1, backoff_max_duration="1s")

@dsl.pipeline
def nested_pipeline_simple(retries: str) -> bool:
    # if successful, check that the number of retries is correct.
    subtask = print_op(text='test')
    if retries != '2':
        raise Exception('Number of retries has not reached two yet.')
    return True

@dsl.pipeline()
def retry_pipeline():
    task1 = verify_retries(retries="{{retries}}").set_retry(num_retries=2)
    task2 = print_op(text='test').set_retry(num_retries=2, backoff_duration="1s", backoff_factor=1, backoff_max_duration="1s")
    task3 = print_op(text='test').set_retry(num_retries=0)
    nested_pipeline()

    #handle the above arguments at a pipeline level.
    task5 = nested_pipeline_simple().set_retry(num_retries=2)
    task6 = nested_pipeline_simple().set_retry(num_retries=2, backoff_duration="1s", backoff_factor=1, backoff_max_duration="1s")
    task7 = nested_pipeline().set_retry(num_retries=0)

if __name__ == '__main__':
    compiler.Compiler.compile(
        pipeline_func=retry_pipeline,
        package_path='pipeline_with_retry.yaml')

