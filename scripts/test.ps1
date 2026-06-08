# Test runner for Windows (PowerShell). Usage: .\scripts\test.ps1 <target>
# Examples: .\scripts\test.ps1 unit | coverage | integration | all

param(
    [Parameter(Position = 0)]
    [ValidateSet("unit", "coverage", "integration", "idempotency", "security", "replay", "observability", "failure", "concurrency", "chaos", "all")]
    [string]$Target = "unit"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

function Invoke-GoTest {
    param([string[]]$GoArgs)
    & go @GoArgs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

switch ($Target) {
    "unit" {
        Invoke-GoTest @("test", "-count=1", "./internal/...", "./pkg/...")
    }
    "coverage" {
        Invoke-GoTest @("test", "-coverprofile=coverage.out", "./internal/...", "./pkg/...")
        & go tool cover -func="coverage.out"
    }
    "integration" {
        Invoke-GoTest @("test", "-tags=integration", "-timeout=20m", "-count=1", "./tests/integration/...")
    }
    "idempotency" {
        Invoke-GoTest @("test", "-tags=integration", "-timeout=20m", "-count=1", "./tests/idempotency/...")
    }
    "security" {
        Invoke-GoTest @("test", "-tags=security", "-timeout=15m", "-count=1", "./tests/security/...")
    }
    "replay" {
        Invoke-GoTest @("test", "-tags=integration", "-timeout=30m", "-count=1", "./tests/replay/...")
    }
    "observability" {
        Invoke-GoTest @("test", "-tags=integration", "-timeout=15m", "-count=1", "./tests/observability/...")
    }
    "failure" {
        Invoke-GoTest @("test", "-tags=failure", "-timeout=20m", "-count=1", "./tests/failure/...")
    }
    "concurrency" {
        Invoke-GoTest @("test", "-tags=concurrency", "-timeout=30m", "-count=1", "./tests/concurrency/...")
    }
    "chaos" {
        $env:RUN_CHAOS = "1"
        Invoke-GoTest @("test", "-tags=chaos", "-timeout=20m", "-count=1", "./tests/chaos/...")
    }
    "all" {
        & $PSCommandPath unit
        & $PSCommandPath integration
        & $PSCommandPath idempotency
        & $PSCommandPath security
        & $PSCommandPath replay
        & $PSCommandPath observability
    }
}
