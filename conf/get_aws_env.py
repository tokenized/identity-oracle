import boto3
import json
import os
import sys
from botocore.exceptions import ClientError


def output_exports(env_dict):
    try:
        # if it's a JSON string - parse it
        env_dict = json.loads(env_dict)
    except TypeError:
        pass
    print("\n".join(["export {}={}".format(
        key,
        value
    ) for key, value in env_dict.items()]))


def load_variables():
    ENV_MAP = {
        "prod": "identityoracle-config",
        "test": "identityoracle-config",
        "dev": "identityoracle-config",
        "default": "identityoracle-config",
    }

    try:
        env_id = sys.argv[1].replace('"', '')
    except IndexError:
        env_id = "default"

    secret_name = ENV_MAP[env_id]
    region_name = "eu-west-2"

    # Check for the injected ENV store from AWS Secrets Manager
    NEXUS_API_ENV = os.environ.get('IDENTITYORACLE_ENV', None)
    if NEXUS_API_ENV is not None:
        output_exports(NEXUS_API_ENV)
        return

    session = boto3.session.Session(
        aws_access_key_id=os.environ.get('AWS_ACCESS_KEY_ID'),
        aws_secret_access_key=os.environ.get('AWS_SECRET_ACCESS_KEY'),
    )
    client = session.client(
        service_name='secretsmanager',
        region_name=region_name,
    )

    try:
        get_secret_value_response = client.get_secret_value(
            SecretId=secret_name
        )
    except ClientError as e:
        if e.response['Error']['Code'] == 'DecryptionFailureException':
            # Secrets Manager can't decrypt the protected secret text using the provided KMS key.
            raise e
        elif e.response['Error']['Code'] == 'InternalServiceErrorException':
            # An error occurred on the server side.
            raise e
        elif e.response['Error']['Code'] == 'InvalidParameterException':
            # You provided an invalid value for a parameter.
            raise e
        elif e.response['Error']['Code'] == 'InvalidRequestException':
            # You provided a parameter value that is not valid for the current state of the resource.
            raise e
        elif e.response['Error']['Code'] == 'ResourceNotFoundException':
            # We can't find the resource that you asked for.
            raise e
        elif e.response['Error']['Code'] == 'UnrecognizedClientException':
            # Invalid AWS Key ID / Secret
            raise e
        else:
            raise e
    else:
        # Decrypts secret using the associated KMS CMK.
        if 'SecretString' in get_secret_value_response:
            secret = json.loads(
                get_secret_value_response['SecretString']
            )
            output_exports(secret)


load_variables()
