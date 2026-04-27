@echo off
setlocal enabledelayedexpansion

:: 检查domain_monitor.exe是否运行
tasklist | findstr "domain_monitor.exe" >nul && (
    echo shutdwon domain_monitor.exe...
    taskkill /f /im domain_monitor.exe
    
    :: 检查日志文件是否存在
    if exist "domain_block_monitor.log" (
        :: 生成时间戳格式：YYYYMMDD_HHMMSS
        for /f "tokens=1-3 delims=/ " %%a in ('date /t') do set datepart=%%c%%a%%b
        for /f "tokens=1-3 delims=:." %%a in ('time /t') do set timepart=%%a%%b%%c
        
        :: 备份日志文件
        set "timestamp=!datepart!_!timepart!"
        set "backup_file=domain_block_monitor_!timestamp!.log"
        
        copy "domain_block_monitor.log" "!backup_file!" >nul
        if exist "!backup_file!" (
            del "domain_block_monitor.log"
        ) else (
            echo "backup file failed"
        )
    ) else (
        echo "not find domain_block_monitor.log file"
    )
    
    echo "alredy stop domain_monitor.exe and delete log file"
) || (
    echo "not find "domain_monitor.exe" process"
)
timeout /t 10 /nobreak
exit