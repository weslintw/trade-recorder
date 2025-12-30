@echo off
cd frontend
echo Installing dependencies...
call pnpm install
echo Building...
set NODE_OPTIONS=--max-old-space-size=4096
call pnpm run build
if %ERRORLEVEL% NEQ 0 (
  echo Build failed!
  exit /b %ERRORLEVEL%
)
echo Build success!
