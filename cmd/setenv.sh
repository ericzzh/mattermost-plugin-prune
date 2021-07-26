#!/usr/bin/env bash

        # "DriverName": "mysql",
        # "DataSource": "mmuser:mostest@tcp(localhost:3306)/mattermost_test?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s\u0026multiStatements=true",

export MM_SERVER_PATH=~/go/src/mattermost-server
export MM_SQLSETTINGS_DRIVERNAME=mysql
export MM_CONFIG=./config.json

export MM_SQLSETTINGS_DRIVERNAME=mysql
echo "**********ZZH MOD*********  Set MM_SQLSETTINGS_DRIVERNAME: $MM_SQLSETTINGS_DRIVERNAME" 
export MM_SQLSETTINGS_DATASOURCE="mmuser:mostest@tcp(localhost:3306)/mattermost_test?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s\u0026multiStatements=true"
echo "***********ZZH MOD*********  Set MM_SQLSETTINGS_DATASOURCE: $MM_SQLSETTINGS_DATASOURCE" 
export MM_SERVICESETTINGS_ENABLELOCALMODE=true
echo "***********ZZH MOD*********  Set MM_SERVICESETTINGS_ENABLELOCALMODE: $MM_SERVICESETTINGS_ENABLELOCALMODE" 

