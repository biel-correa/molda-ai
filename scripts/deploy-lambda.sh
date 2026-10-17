#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# MoldaAí — AWS Lambda Deployment Automation Script
#
# Supports:
# - Continuous updates of existing Lambda functions
# - Automated initial provisioning (IAM Role, Lambda Function, Function URL, CORS)
# ==============================================================================

FUNCTION_NAME="${LAMBDA_FUNCTION_NAME:-molda-ai}"
REGION="${AWS_REGION:-us-east-1}"
ROLE_NAME="${LAMBDA_ROLE_NAME:-${FUNCTION_NAME}-exec-role}"
MEMORY_SIZE="${MEMORY_SIZE:-512}"
TIMEOUT="${TIMEOUT:-15}"

# Resolve repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ZIP_PATH="${REPO_ROOT}/app/moldaai-lambda.zip"

echo "=========================================================="
echo "🚀 Deploying MoldaAí to AWS Lambda"
echo "   Function:     ${FUNCTION_NAME}"
echo "   Region:       ${REGION}"
echo "   Architecture: ARM64 (Graviton)"
echo "   Runtime:      provided.al2023"
echo "=========================================================="

# 1. Ensure deployment zip exists
if [ ! -f "${ZIP_PATH}" ]; then
  echo "📦 Deployment zip not found at ${ZIP_PATH}. Compiling package..."
  (cd "${REPO_ROOT}/app" && make build-lambda)
fi

echo "✅ Package ready: ${ZIP_PATH} ($(du -h "${ZIP_PATH}" | cut -f1))"

# 2. Check if function exists
echo "🔍 Checking if Lambda function '${FUNCTION_NAME}' exists..."
if aws lambda get-function --function-name "${FUNCTION_NAME}" --region "${REGION}" >/dev/null 2>&1; then
  echo "🔄 Function exists. Updating code..."
  aws lambda update-function-code \
    --function-name "${FUNCTION_NAME}" \
    --zip-file "fileb://${ZIP_PATH}" \
    --region "${REGION}" >/dev/null

  echo "⏳ Waiting for code update to complete..."
  aws lambda wait function-updated \
    --function-name "${FUNCTION_NAME}" \
    --region "${REGION}"

  echo "⚙️ Updating function configuration (Handler: bootstrap, RAM: ${MEMORY_SIZE}MB, Timeout: ${TIMEOUT}s)..."
  aws lambda update-function-configuration \
    --function-name "${FUNCTION_NAME}" \
    --handler "bootstrap" \
    --memory-size "${MEMORY_SIZE}" \
    --timeout "${TIMEOUT}" \
    --environment "Variables={GIN_MODE=release}" \
    --region "${REGION}" >/dev/null

  aws lambda wait function-updated \
    --function-name "${FUNCTION_NAME}" \
    --region "${REGION}"
else
  echo "🆕 Function '${FUNCTION_NAME}' does not exist. Initializing new Lambda resource..."

  # Check or create IAM Role
  ROLE_ARN=""
  if aws iam get-role --role-name "${ROLE_NAME}" >/dev/null 2>&1; then
    ROLE_ARN=$(aws iam get-role --role-name "${ROLE_NAME}" --query "Role.Arn" --output text)
    echo "🔑 Using existing IAM Role: ${ROLE_ARN}"
  else
    echo "🔑 Creating IAM Role '${ROLE_NAME}'..."
    TRUST_POLICY='{
      "Version": "2012-10-17",
      "Statement": [
        {
          "Effect": "Allow",
          "Principal": {
            "Service": "lambda.amazonaws.com"
          },
          "Action": "sts:AssumeRole"
        }
      ]
    }'

    ROLE_ARN=$(aws iam create-role \
      --role-name "${ROLE_NAME}" \
      --assume-role-policy-document "${TRUST_POLICY}" \
      --query "Role.Arn" \
      --output text)

    echo "📜 Attaching AWSLambdaBasicExecutionRole policy..."
    aws iam attach-role-policy \
      --role-name "${ROLE_NAME}" \
      --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"

    echo "⏳ Waiting 10 seconds for IAM Role propagation..."
    sleep 10
  fi

  # Create Function
  echo "⚡ Creating Lambda function '${FUNCTION_NAME}'..."
  aws lambda create-function \
    --function-name "${FUNCTION_NAME}" \
    --runtime "provided.al2023" \
    --architectures "arm64" \
    --role "${ROLE_ARN}" \
    --handler "bootstrap" \
    --zip-file "fileb://${ZIP_PATH}" \
    --memory-size "${MEMORY_SIZE}" \
    --timeout "${TIMEOUT}" \
    --region "${REGION}" >/dev/null

  echo "⏳ Waiting for function to become active..."
  aws lambda wait function-active-v2 \
    --function-name "${FUNCTION_NAME}" \
    --region "${REGION}" 2>/dev/null || sleep 5
fi

# 3. Ensure Function URL is configured
echo "🌐 Configuring Lambda Function URL..."
FUNCTION_URL=""
if aws lambda get-function-url-config --function-name "${FUNCTION_NAME}" --region "${REGION}" >/dev/null 2>&1; then
  FUNCTION_URL=$(aws lambda get-function-url-config \
    --function-name "${FUNCTION_NAME}" \
    --region "${REGION}" \
    --query "FunctionUrl" \
    --output text)
else
  echo "🔗 Creating new public Function URL..."
  FUNCTION_URL=$(aws lambda create-function-url-config \
    --function-name "${FUNCTION_NAME}" \
    --auth-type "NONE" \
    --cors '{"AllowOrigins":["*"],"AllowMethods":["GET","POST"],"AllowHeaders":["*"],"MaxAge":86400}' \
    --region "${REGION}" \
    --query "FunctionUrl" \
    --output text)

  echo "🛡️ Granting public invocation permissions..."
  aws lambda add-permission \
    --function-name "${FUNCTION_NAME}" \
    --action "lambda:InvokeFunctionUrl" \
    --principal "*" \
    --function-url-auth-type "NONE" \
    --statement-id "FunctionURLAllowPublicAccess" \
    --region "${REGION}" >/dev/null 2>&1 || true

  aws lambda add-permission \
    --function-name "${FUNCTION_NAME}" \
    --action "lambda:InvokeFunction" \
    --principal "*" \
    --statement-id "FunctionPublicInvoke" \
    --region "${REGION}" >/dev/null 2>&1 || true
fi

# Save URL for CI/CD summaries
echo "${FUNCTION_URL}" > "${REPO_ROOT}/.deployment_url"

echo ""
echo "=========================================================="
echo "🎉 DEPLOYMENT SUCCESSFUL!"
echo "   Endpoint URL: ${FUNCTION_URL}"
echo "=========================================================="
