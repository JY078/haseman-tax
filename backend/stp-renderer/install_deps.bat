@echo off
chcp 65001 >nul
echo ========================================
echo STP Renderer 依赖安装脚本
echo ========================================
echo.

echo [1/2] 正在安装 pythonocc-core...
pip install pythonocc-core>=0.18.2
if errorlevel 1 (
    echo pythonocc-core 安装失败，尝试从 conda-forge 安装...
    conda install -c conda-forge pythonocc-core -y
)

echo.
echo [2/2] 正在安装其他依赖...
pip install open3d>=0.17.0 numpy>=1.24.0 Pillow>=10.0.0 oss2>=2.18.0 redis>=5.0.0 requests>=2.31.0 matplotlib>=3.7.0 trimesh>=4.0.0 pyassimp>=5.2.5

echo.
echo ========================================
echo 安装完成！验证安装...
echo ========================================
python -c "import OCC; print('✓ pythonocc-core OK')" 2>nul || echo "✗ pythonocc-core 未安装"
python -c "import open3d; print('✓ open3d OK')" 2>nul || echo "✗ open3d 未安装"

echo.
echo 请在 backend/.env 中确认配置：
echo   STP_WORKER_PYTHON=C:/Users/HSMY-SJ-2/miniconda3/python.exe
echo   STP_WORKER_PATH=D:/workplace/HaiErMan/backend/stp-renderer
echo.
pause
