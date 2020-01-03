#! /bin/bash

# Example usage
# source conf/env.sh --environment=test && make
# expects to use secret name = clientsd-env  


# environment options: "prod", "test"
for i in "$@"
do
case $i in
    -e=*|--environment=*)
    ENVIRONMENT=${i#*=}
    shift
    ;;
    *)
    ;;
esac
done
# Execute "export" statements from the Python script
# retrieving secrets dynamically from AWS Secrets Manager
echo "Retrieving AWS Secrets for: $ENVIRONMENT"
eval $(python3 conf/get_aws_env.py $ENVIRONMENT)
