![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/yonahd/kor)
![GitHub release (with filter)](https://img.shields.io/github/v/release/yonahd/kor?color=green&link=https://github.com/yonahd/kor/releases)

# Kor - Kubernetes Orphaned Resources Finder

Kor is a tool to discover unused Kubernetes resources. Currently, Kor can identify and list unused:
- ConfigMaps  
- Secrets.
- Services
- ServiceAccounts
- Deployments
- Statefulsets
- Roles
- Hpas
- Pvcs
- Ingresses
- Pdbs

![Kor Screenshot](/images/screenshot.png)

## Installation

Download the binary for your operating system from the [releases page](https://github.com/yonahd/kor/releases) and add it to your system's PATH.

For MacOS users, you can install Kor using Homebrew:
```sh
brew install kor
```

## Usage

Kor provides various subcommands to identify and list unused resources. The available commands are:

- `all`: Gets all unused resources for the specified namespace or all namespaces.
- `configmap`: Gets unused configmaps for the specified namespace or all namespaces.
- `secret`: Gets unused secrets for the specified namespace or all namespaces.
- `services`: Gets unused services for the specified namespace or all namespaces.
- `serviceaccount`: Gets unused service accounts for the specified namespace or all namespaces.
- `deployments`: Gets unused service accounts for the specified namespace or all namespaces.
- `statefulsets`: Gets unused service accounts for the specified namespace or all namespaces.
- `role`: Gets unused roles for the specified namespace or all namespaces.
- `hps`: Gets unused hpa for the specified namespace or all namespaces.
- `pvc`: Gets unused pvcs for the specified namespace or all namespaces.
- `ingress`: Gets unused ingresses for the specified namespace or all namespaces.
- `pdb`: Gets unused pdbs for the specified namespace or all namespaces.

### Supported Flags
```
-h, --help                help for role
-k, --kubeconfig string   Path to kubeconfig file (optional)
-n, --namespace string    Namespace to run on
--output string       Output format (table, json or yaml) (default "table")
```

To use a specific subcommand, run `kor [subcommand] [flags]`.

```sh
kor all --namespace my-namespace
```

For more information about each subcommand and its available flags, you can use the `--help` flag.

```sh
kor [subcommand] --help
```

## Supported resources and limitations

| Resource        | What it looks for                                                                                                                                                                                                                  | Known False Positives  ⚠️                                                                                                    |
|-----------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------|
| Configmaps      | Configmaps not used in the following places:<br/>- Pods<br/>- Containers <br/>- Configmaps used through volumes <br/>- Configmaps used through environment variables                                                               | Configmaps used by resources which don't explicitly state them in the config.<br/> e.g Grafana dashboards loaded dynamically opa policies fluentd configs |
| Secrets         | Secrets not used in the following places:<br/>- Pods<br/>- Containers <br/>- Secrets used through volumes <br/>- Secrets used through environment variables<br/>- Secrets used by ingress TLS<br/>-Secrets used by ServiceAccounts |    Secrets used by resources which don't explicitly state them in the config                                                                                                                         |
| Services        | Services with no endpoints                                                                                                                                                                                                         |                                                                                                                              |
| Deployments     | Deployments with 0 Replicas                                                                                                                                                                                                        |                                                                                                                              |
| ServiceAccounts | ServiceAccounts unused by pods<br/>ServiceAccounts unused by roleBinding or clusterRoleBinding                                                                                                                                     |                                                                                                                              |
| Statefulsets    | Statefulsets with 0 Replicas                                                                                                                                                                                                       |                                                                                                                              |
| Roles           | Roles not used in roleBinding                                                                                                                                                                                                      |                                                                                                                              |
| Pvcs            | Pvcs not used in pods                                                                                                                                                                                                              |                                                                                                                              |
| Ingresses       | Ingresses not pointing at any service.                                                                                                                                                                                             |                                                                                                                              |
| Hpas            | Hpas not used in Deployments   <br/>    Hpas not used in Statefulsets                                                                                                                                                              |                                                                                                                              |
| Pdbs            | Pdbs not used in Deployments   <br/>    Pdbs not used in Statefulsets                                                                                                                                                              |                                                                                                                              |



## Import Option
You can also use kor as a Go library to programmatically discover unused resources. By importing the github.com/yonahd/kor/pkg/kor package, you can call the relevant functions to retrieve unused resources. The library provides the option to get the results in JSON format by specifying the outputFormat parameter.

```go
import (
"github.com/yonahd/kor/pkg/kor"
)

func main() {
namespace := "my-namespace"
outputFormat := "json" // Set to "json" for JSON output

    if outputFormat == "json" {
        jsonResponse, err := kor.GetUnusedDeploymentsJSON(namespace, kubeconfig)
        if err != nil {
            // Handle error
        }
        // Process the JSON response
        // ...
    } else {
        kor.GetUnusedDeployments(namespace)
    }
}
```


## Contributing

Contributions are welcome! If you encounter any bugs or have suggestions for improvements, please open an issue in the [issue tracker](https://github.com/yonahd/kor/issues).

## License

This project is open-source and available under the [MIT License](LICENSE). Feel free to use, modify, and distribute it as per the terms of the license.

