# Runner post-job hook. Each runner has its own Windows account, so its TEMP
# holds only what this runner's jobs left.
Get-ChildItem -Force -LiteralPath $env:TEMP -ErrorAction SilentlyContinue |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
exit 0
