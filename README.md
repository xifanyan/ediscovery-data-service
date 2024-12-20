# ediscovery-data-service

## Getting started

- Download and Install Golang [https://go.dev/doc/install]
- Clone the repo

    ```Command Prompt
    git clone https://github.com/xifanyan/ediscovery-data-service
    ```

- Build the binary (bin/ediscovery-data-service.exe)

    ```Command Prompt
    cd ediscovery-data-service
    .\build.bat
    ```

- Run (make sure config.json is in the same directory)

    ```Command Prompt
    .\bin\ediscovery-data-service.exe
    ```

    or

    ```Command Prompt
    go run main.go
    ```

- Install service as Windows Service with Administrator Privilege using NSSM [https://nssm.cc]

    ```Command Prompt (Administrator Privilege)
    nssm install eDiscoveryDataService
    ```

    ![alt text](nssm_config.png?raw=true "NSSM Configuration")
    > Note: make sure Startup directory has the config.json file.

- Uninstall eDiscoveryDataService

    ```Command Prompt (Administrator Privilege)
    nssm remove eDiscoveryDataService
    ```

## APIs

- [reference](api.http)
