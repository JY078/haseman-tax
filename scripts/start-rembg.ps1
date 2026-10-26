param([int]$Port = 8011)
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$service = Join-Path $repo 'backend/rembg-service'
$python = Join-Path $repo '.cache/rembg-venv/Scripts/python.exe'
$cache = Join-Path $repo '.cache'
New-Item -ItemType Directory -Force -Path $cache | Out-Null
try {
    $health = Invoke-RestMethod "http://127.0.0.1:$Port/health" -TimeoutSec 3
    if ($health.model -eq 'birefnet-general') { Write-Host "rembg ready: http://127.0.0.1:$Port"; exit 0 }
} catch { }
if (Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue) {
    throw "Port $Port is occupied by another service. Set -Port and the backend REMBG_SERVICE_URL to the same address."
}
if (!(Test-Path $python)) { python -m venv (Join-Path $cache 'rembg-venv'); if ($LASTEXITCODE) { throw 'venv creation failed' } }
& $python -m pip install -r (Join-Path $service 'requirements.txt')
if ($LASTEXITCODE) { throw 'Dependency installation failed' }
$env:OMP_NUM_THREADS = '3'
$env:NUMBA_NUM_THREADS = '3'
$env:OPENBLAS_NUM_THREADS = '3'
$env:MKL_NUM_THREADS = '3'
$env:U2NET_HOME = Join-Path $cache 'rembg-models'
$process = Start-Process -FilePath $python -ArgumentList @('-m', 'uvicorn', 'app:app', '--host', '127.0.0.1', '--port', "$Port", '--workers', '1') -WorkingDirectory $service -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $cache 'rembg.out.log') -RedirectStandardError (Join-Path $cache 'rembg.err.log')
$process.PriorityClass = 'BelowNormal'
# Restrict the development process to three of its permitted logical processors.
$allowed = $process.ProcessorAffinity.ToInt64()
$selected = [long]0
$count = 0
for ($bit = 0; $bit -lt 63 -and $count -lt 3; $bit++) {
    $mask = [long]1 -shl $bit
    if ($allowed -band $mask) { $selected = $selected -bor $mask; $count++ }
}
$process.ProcessorAffinity = [IntPtr]$selected
# A Windows venv launcher may already have spawned the actual interpreter.
Get-CimInstance Win32_Process -Filter "ParentProcessId=$($process.Id)" | ForEach-Object {
    $child = Get-Process -Id $_.ProcessId -ErrorAction SilentlyContinue
    if ($child) { $child.PriorityClass = 'BelowNormal'; $child.ProcessorAffinity = [IntPtr]$selected }
}
Write-Host "rembg starting (PID $($process.Id)); health: http://127.0.0.1:$Port/health"
Write-Host "Logs: $cache/rembg.err.log. For production memory isolation, use Docker Compose."
