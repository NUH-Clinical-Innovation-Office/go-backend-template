#!/bin/bash

# CI/CD setup for repositories created from go-backend-template.
#
# Generates .github/workflows/ wired to the shared reusable workflows in
# NUH-Clinical-Innovation-Office/ci-workflows, matching the setup that is
# already running in NUH-Clinical-Innovation-Office/pre-consult-backend.
#
# The template ships self-contained `reusable-*.yml` workflows. Those duplicate
# what the shared repository already provides, so this script replaces them.
# Anything it overwrites is moved to .github/workflows-backup/ first.

set -euo pipefail

RED=$'\033[0;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
BLUE=$'\033[0;34m'
NC=$'\033[0m'

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Repo root, regardless of where the script is invoked from.
REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$REPO_ROOT"

WORKFLOW_DIR=".github/workflows"
BACKUP_DIR=".github/workflows-backup"

# Reusable workflows are pinned to a tag, never to a moving branch.
CI_WORKFLOWS_REF="v1"
CI_WORKFLOWS="NUH-Clinical-Innovation-Office/ci-workflows"

# Reports whether the Makefile defines the given target.
has_make_target() {
    local target=$1
    [ -f "Makefile" ] && grep -qE "^${target}:" Makefile
}

# Prompts with a default, echoing the answer.
prompt_with_default() {
    local prompt_text=$1
    local default_value=$2
    local answer
    read -r -p "${prompt_text} (default: ${default_value}): " answer
    echo "${answer:-$default_value}"
}

prompt_yes_no() {
    local prompt_text=$1
    local answer
    read -r -p "${prompt_text} (y/N): " answer
    [[ "$answer" =~ ^[Yy]$ ]]
}

require_prerequisites() {
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Run this from the repository root of a Go module."
        exit 1
    fi

    if [ ! -f "Dockerfile" ]; then
        print_warning "No Dockerfile found. The image build job will fail until one exists."
    fi
}

# Moves existing workflows aside so nothing is silently destroyed.
backup_existing_workflows() {
    if [ ! -d "$WORKFLOW_DIR" ] || [ -z "$(ls -A "$WORKFLOW_DIR" 2>/dev/null)" ]; then
        return 0
    fi

    print_warning "Existing workflows found in ${WORKFLOW_DIR}:"
    ls -1 "$WORKFLOW_DIR" | sed 's/^/    /'
    echo ""

    if ! prompt_yes_no "Move these to ${BACKUP_DIR} and replace them?"; then
        print_warning "Setup cancelled. No files were changed."
        exit 0
    fi

    mkdir -p "$BACKUP_DIR"
    # `mv` per-file so a re-run does not fail on an existing backup dir.
    for file in "$WORKFLOW_DIR"/*; do
        [ -e "$file" ] || continue
        mv -f "$file" "$BACKUP_DIR/"
    done
    print_success "Backed up previous workflows to ${BACKUP_DIR}"
}

# Single ci.yml covering both push and pull_request, matching
# pre-consult-backend. Jobs that publish or deploy are gated on push-to-main so
# pull requests run checks only.
write_ci_workflow() {
    cat > "${WORKFLOW_DIR}/ci.yml" <<YAML
name: CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: \${{ github.repository }}

jobs:
  ci:
    name: CI (Lint, Unit Test, Build)
    permissions:
      contents: read
    uses: ${CI_WORKFLOWS}/.github/workflows/build-go.yml@${CI_WORKFLOWS_REF}
    with:
      runner: ${RUNNER}
      gh_private: ${GH_PRIVATE}
    secrets: inherit
YAML

    if [ "$ENABLE_OPENAPI_CHECK" = "true" ]; then
        # Fails the build if committed generated code has drifted from the spec.
        cat >> "${WORKFLOW_DIR}/ci.yml" <<YAML

  openapi-check:
    name: OpenAPI Contract Check
    runs-on: ubuntu-latest
    timeout-minutes: 10
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
      - name: Verify generated API code is up to date
        run: |
          go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION}
          make openapi
          git diff --exit-code ${GENERATED_API_DIR} || (echo "${GENERATED_API_DIR} is stale; run 'make openapi' and commit" && exit 1)
      - name: Lint OpenAPI spec
        run: npx -y @redocly/cli@${REDOCLY_VERSION} lint ${OPENAPI_SPEC}
YAML
    fi

    if [ "$ENABLE_INTEGRATION" = "true" ]; then
        # Integration tests use testcontainers, which needs a working Docker
        # daemon; the dind service supplies one.
        cat >> "${WORKFLOW_DIR}/ci.yml" <<YAML

  test-integration:
    name: Integration Tests
    runs-on: ubuntu-latest
    timeout-minutes: 20
    permissions:
      contents: read
    services:
      docker:
        image: docker:dind
        options: --privileged
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
      - name: Run integration tests
        run: make test-integration
        env:
          TESTCONTAINERS_RYUK_DISABLED: "false"
YAML
    fi

    # Publish depends on every gate that is actually present.
    local publish_needs="ci"
    [ "$ENABLE_OPENAPI_CHECK" = "true" ] && publish_needs="${publish_needs}, openapi-check"
    [ "$ENABLE_INTEGRATION" = "true" ] && publish_needs="${publish_needs}, test-integration"

    cat >> "${WORKFLOW_DIR}/ci.yml" <<YAML

  docker-publish:
    name: Build & Push Docker Image
    needs: [${publish_needs}]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    permissions:
      contents: read
      packages: write
    uses: ${CI_WORKFLOWS}/.github/workflows/docker-build-push.yml@${CI_WORKFLOWS_REF}
    with:
      platforms: ${PLATFORMS}
      runner: ${RUNNER}
    secrets: inherit

  security-scan-image:
    name: Security Scan (Image)
    needs: docker-publish
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    permissions:
      contents: read
      packages: read
      # Central security-scan.yml declares security-events: write for its SARIF
      # path; GitHub requires the caller to grant it even when upload_sarif:false.
      security-events: write
    uses: ${CI_WORKFLOWS}/.github/workflows/security-scan.yml@${CI_WORKFLOWS_REF}
    with:
      image_tag: \${{ needs.docker-publish.outputs.image_tag }}
      severity: CRITICAL
      runner: ${RUNNER}
      trivy_platform: ${TRIVY_PLATFORM}
      upload_sarif: false
    secrets: inherit
YAML

    if [ "$ENABLE_DEPLOY" = "true" ]; then
        # Deploy consumes versioned_tag (bare branch-sha), not image_tag (full
        # registry reference), because the Helm chart takes a bare tag.
        cat >> "${WORKFLOW_DIR}/ci.yml" <<YAML

  deploy-development:
    name: Deploy to Development
    needs: [docker-publish, security-scan-image]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    permissions:
      contents: read
      packages: read
      id-token: write
    uses: ./.github/workflows/deploy.yml
    with:
      environment: development
      image_tag: \${{ needs.docker-publish.outputs.versioned_tag }}
    secrets:
      AWS_DEV_DEPLOY_ROLE_ARN: \${{ secrets.AWS_DEV_DEPLOY_ROLE_ARN }}
      IRSA_ROLE_ARN: \${{ secrets.IRSA_ROLE_ARN }}
YAML
    fi

    print_success "Created ${WORKFLOW_DIR}/ci.yml"
}

write_deploy_workflow() {
    cat > "${WORKFLOW_DIR}/deploy.yml" <<YAML
name: Deploy to EKS Development

on:
  workflow_dispatch:
    inputs:
      image_tag:
        description: "Immutable versioned image tag to deploy"
        required: true
        type: string
  workflow_call:
    inputs:
      environment:
        required: false
        type: string
        default: development
      image_tag:
        description: "Immutable versioned image tag to deploy"
        required: true
        type: string
    secrets:
      AWS_DEV_DEPLOY_ROLE_ARN:
        required: true
      IRSA_ROLE_ARN:
        required: false

jobs:
  deploy:
    name: Deploy to development
    permissions:
      contents: read
      packages: read
      id-token: write
    uses: ${CI_WORKFLOWS}/.github/workflows/deploy-eks-helm.yml@${CI_WORKFLOWS_REF}
    with:
      image_tag: \${{ inputs.image_tag }}
      environment_name: development
      aws_region: ${AWS_REGION}
      eks_cluster_name: ${EKS_CLUSTER}
      namespace: ${K8S_NAMESPACE}
      helm_chart_path: ${HELM_CHART_PATH}
      helm_values_file: ${HELM_VALUES_FILE}
      chart_release_name: ${RELEASE_NAME}
      required_rollout_deployments: ${RELEASE_NAME}
    secrets:
      AWS_DEV_DEPLOY_ROLE_ARN: \${{ secrets.AWS_DEV_DEPLOY_ROLE_ARN }}
      IRSA_ROLE_ARN: \${{ secrets.IRSA_ROLE_ARN }}
YAML
    print_success "Created ${WORKFLOW_DIR}/deploy.yml"
}

write_image_cleanup_workflow() {
    cat > "${WORKFLOW_DIR}/image-cleanup.yml" <<YAML
name: Container Image Cleanup

on:
  schedule:
    # Run every Sunday at 2 AM UTC
    - cron: "0 2 * * 0"
  workflow_dispatch: # Allow manual trigger

jobs:
  cleanup:
    name: Cleanup Old Container Images
    permissions:
      packages: write
      contents: read
    uses: ${CI_WORKFLOWS}/.github/workflows/image-cleanup.yml@${CI_WORKFLOWS_REF}
    with:
      keep_latest_production: 3
    secrets: inherit
YAML
    print_success "Created ${WORKFLOW_DIR}/image-cleanup.yml"
}

print_next_steps() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║                    CI Setup Complete! 🎉                   ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    print_info "Generated workflows:"
    echo "    ${WORKFLOW_DIR}/ci.yml             - push + PR pipeline"
    if [ "$ENABLE_DEPLOY" = "true" ]; then
        echo "    ${WORKFLOW_DIR}/deploy.yml         - AWS EKS deploy"
    fi
    if [ "$ENABLE_CLEANUP" = "true" ]; then
        echo "    ${WORKFLOW_DIR}/image-cleanup.yml  - weekly GHCR pruning"
    fi
    echo ""
    print_info "Before the first run, confirm in GitHub:"
    echo "    1. Settings > Actions > General > Workflow permissions"
    echo "       allows access to the ${CI_WORKFLOWS} repository."
    if [ "$GH_PRIVATE" = "true" ]; then
        echo "    2. Repository secret GH_PAT is set (private Go module access)."
    fi
    if [ "$ENABLE_DEPLOY" = "true" ]; then
        echo "    3. Repository secret AWS_DEV_DEPLOY_ROLE_ARN is set to the"
        echo "       OIDC role ARN that GitHub Actions assumes."
        echo "    4. Repository secret IRSA_ROLE_ARN is set if the workload needs"
        echo "       its own ServiceAccount IAM role (optional)."
        echo "    5. A 'development' environment exists (Settings > Environments)."
        echo "    6. The Helm chart at ${HELM_CHART_PATH} has a values file at"
        echo "       ${HELM_VALUES_FILE} and a deployment named ${RELEASE_NAME}."
    fi
    echo ""
    if [ -d "$BACKUP_DIR" ]; then
        print_warning "Previous workflows are in ${BACKUP_DIR} — delete once you are happy."
    fi
}

main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║           Go CI/CD Setup — Shared Workflows                ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""

    require_prerequisites

    local module_path default_name
    module_path=$(awk '/^module /{print $2; exit}' go.mod)
    default_name=$(basename "$module_path")

    PROJECT_NAME=$(prompt_with_default "Project name" "$default_name")
    if [[ ! "$PROJECT_NAME" =~ ^[a-z0-9-]+$ ]]; then
        print_error "Project name must contain only lowercase letters, numbers, and hyphens"
        exit 1
    fi
    echo ""

    local arch
    arch=$(prompt_with_default "Build architecture [arm64/amd64]" "arm64")
    case "$arch" in
        arm64)
            PLATFORMS="linux/arm64"
            TRIVY_PLATFORM="linux/arm64"
            RUNNER="ubuntu-24.04-arm"
            ;;
        amd64)
            PLATFORMS="linux/amd64"
            TRIVY_PLATFORM="linux/amd64"
            RUNNER="ubuntu-latest"
            ;;
        *)
            print_error "Architecture must be 'arm64' or 'amd64'"
            exit 1
            ;;
    esac
    print_success "Building ${PLATFORMS} on ${RUNNER}"
    echo ""

    # Private module access is opt-in: it requires a GH_PAT secret, and enabling
    # it without one breaks every job in build-go.yml.
    GH_PRIVATE=false
    if [[ "$module_path" == github.com/NUH-Clinical-Innovation-Office/* ]]; then
        print_info "Module is under the NUH org: ${module_path}"
        if prompt_yes_no "Does it depend on private NUH modules (needs GH_PAT)?"; then
            GH_PRIVATE=true
        fi
    else
        if prompt_yes_no "Enable private Go module access (needs GH_PAT secret)?"; then
            GH_PRIVATE=true
        fi
    fi
    echo ""

    # Integration tests only make sense if the Makefile target exists.
    ENABLE_INTEGRATION=false
    if has_make_target "test-integration"; then
        print_info "Found 'test-integration' target in Makefile"
        if prompt_yes_no "Add an integration test job (docker:dind + testcontainers)?"; then
            ENABLE_INTEGRATION=true
        fi
    else
        print_warning "No 'test-integration' target in Makefile — skipping integration job"
    fi
    echo ""

    # OpenAPI drift check only applies to spec-first repos.
    ENABLE_OPENAPI_CHECK=false
    OAPI_CODEGEN_VERSION="v2.4.1"
    REDOCLY_VERSION="1.25.11"
    OPENAPI_SPEC="api/openapi.yaml"
    GENERATED_API_DIR="internal/api"
    if has_make_target "openapi" && [ -f "$OPENAPI_SPEC" ]; then
        print_info "Found 'openapi' target and ${OPENAPI_SPEC}"
        if prompt_yes_no "Add an OpenAPI codegen drift + spec lint job?"; then
            ENABLE_OPENAPI_CHECK=true
            OPENAPI_SPEC=$(prompt_with_default "  OpenAPI spec path" "$OPENAPI_SPEC")
            GENERATED_API_DIR=$(prompt_with_default "  Generated code directory" "$GENERATED_API_DIR")
        fi
    else
        print_warning "No 'openapi' target or spec found — skipping contract check"
    fi
    echo ""

    ENABLE_DEPLOY=false
    if prompt_yes_no "Add an AWS EKS development deploy job?"; then
        ENABLE_DEPLOY=true
    fi

    if [ "$ENABLE_DEPLOY" = "true" ]; then
        echo ""
        K8S_NAMESPACE=$(prompt_with_default "Kubernetes namespace" "$PROJECT_NAME")
        RELEASE_NAME=$(prompt_with_default "Helm release / deployment name" "$PROJECT_NAME")
        EKS_CLUSTER=$(prompt_with_default "EKS cluster name" "dev")
        AWS_REGION=$(prompt_with_default "AWS region" "ap-southeast-1")

        # Default to the chart that is actually on disk.
        local default_chart="./helm/${PROJECT_NAME}"
        if [ ! -d "helm/${PROJECT_NAME}" ]; then
            local found_chart
            found_chart=$(find helm -maxdepth 2 -name Chart.yaml 2>/dev/null | head -1)
            if [ -n "$found_chart" ]; then
                default_chart="./$(dirname "$found_chart")"
            fi
        fi
        HELM_CHART_PATH=$(prompt_with_default "Helm chart path" "$default_chart")

        if [ ! -d "$HELM_CHART_PATH" ]; then
            print_warning "Chart directory ${HELM_CHART_PATH} does not exist yet."
        fi

        HELM_VALUES_FILE=$(prompt_with_default "Helm values file" "${HELM_CHART_PATH}/values-development.yaml")
        if [ ! -f "$HELM_VALUES_FILE" ]; then
            print_warning "Values file ${HELM_VALUES_FILE} does not exist yet — create it before deploying."
        fi
    fi
    echo ""

    ENABLE_CLEANUP=false
    if prompt_yes_no "Add a weekly GHCR image cleanup workflow?"; then
        ENABLE_CLEANUP=true
    fi

    echo ""
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║                   Configuration Summary                    ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo ""
    echo "  Project:          ${PROJECT_NAME}"
    echo "  Go module:        ${module_path}"
    echo "  Platforms:        ${PLATFORMS}"
    echo "  Runner:           ${RUNNER}"
    echo "  Shared ref:       ${CI_WORKFLOWS}@${CI_WORKFLOWS_REF}"
    echo "  Private modules:  ${GH_PRIVATE}"
    echo "  Integration job:  ${ENABLE_INTEGRATION}"
    echo "  OpenAPI check:    ${ENABLE_OPENAPI_CHECK}"
    echo "  Deploy job:       ${ENABLE_DEPLOY}"
    if [ "$ENABLE_DEPLOY" = "true" ]; then
        echo "  Namespace:        ${K8S_NAMESPACE}"
        echo "  Release:          ${RELEASE_NAME}"
        echo "  Cluster:          ${EKS_CLUSTER} (${AWS_REGION})"
        echo "  Chart:            ${HELM_CHART_PATH}"
        echo "  Values:           ${HELM_VALUES_FILE}"
    fi
    echo "  Image cleanup:    ${ENABLE_CLEANUP}"
    echo ""

    if ! prompt_yes_no "Proceed?"; then
        print_warning "Setup cancelled"
        exit 0
    fi

    echo ""
    backup_existing_workflows
    mkdir -p "$WORKFLOW_DIR"

    write_ci_workflow
    if [ "$ENABLE_DEPLOY" = "true" ]; then
        write_deploy_workflow
    fi
    if [ "$ENABLE_CLEANUP" = "true" ]; then
        write_image_cleanup_workflow
    fi

    print_next_steps
}

main "$@"
