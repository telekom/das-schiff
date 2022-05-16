# History of `cluster-components`

We have recently changed our approach to managing the configuration of the components we deploy on our clusters. Previously we had a complex folder structure that implemented a hierarchy of configuration. More specific configuration could be used to override more generic files. The order was as follows:

* Default - applies to all clusters
* Environments - applied to clusters in one environment (tst, ref and prod)
* Providers - applied to clusters deployed on a specific infrastructure (metal3 or vSphere)
* Network zones - applied to clusters in a specific network segment
* Sites - applied to clusters hostet at a specific site
* Customers - applied to clusters that belong to a internal customer
* Cluster

We used the ignore spec of `GitRepositories` in conjunction with paths in `Kustomizations` to apply folders from the structure below. The `GitRepositories` were always pointing to our `main` branch, so any changes were immediately applied to all of our clusters. While this makes it easy to perform changes, it is very risky when managing lots of clusters in case something goes wrong. Especially when performing changes at the default level. In addition, the folder structure is very complex (see below). It takes a while until you can find your way around, and its quite hard to track where a specific config option is coming from.

We have revised the structure from time to time. At some point we got rid of the `default` folder for example, and duplicated all the config to the environment folders. This allowed us to properly progress all changes through the different environments. We then realised that we need environment folders at every level in order to perform this staging consistently. So we added `tst`, `ref` and `prd` folders everywhere.

While this worked for a while, it also came with its own problems. Most importantly it was very hard to track whether changes went through the previous stage before being applied to the next. Creating diffs between stages was also difficult and would have required custom tooling.

We then thought about using separate repositories for different environments, and merging forward from one to the next. But since configuration can differ between environments, this is also complicated and would require special tooling to create those forward merges, which ignores certain files. And it also leaves another problem unsolved: staged rollouts.

We could just have pinned the `GitRepositories` to specific commits and updated them one after another, but that would also be hard to track, and it also requires upgrading all configuration for one cluster at the same time (or introduce a lot of duplication of `GitRepositories` and make it even harder to track what is applied to which cluster).

This let to the latest approach described in the main README, and to the plan of using helm for components. The components are individually packaged and versioned, and contain all required configuration, with variation based on location, environment or network zone as needed. The versioned components can be upgraded easily by specifying a tag in the `GitRepository` or the desired version in the `HelmRelease`.

```bash
cluster-components
├ customers # Defaults per customers and environments
│ ├ customer_A
│ │ ├ default # Defaults for customer_A for all environments
│ │ │ ├ configmaps
│ │ │ ├ gitrepositories
│ │ │ ├ kustomizations
│ │ │ └ namespaces
│ │ ├ prd # Specific config for customer_A per environment
│ │ ├ ref
│ │ └ tst
│ └ customer_B
│     ...
├ default # General defaults valid for all customers and environments
│ └ components
│   ├ core
│   │ ├ clusterroles
│   │ ├ configmaps
│   │ └ namespaces
│   └ monitoring
│     ├ configmaps
│     ├ grafana
│     │ ├ dashboards
│     │ │ ├ flux-system
│     │ │ ├ kube-system
│     │ │ └ monitoring-system
│     │ └ datasources
│     ├ namespaces
│     ├ prometheus
│     │ ├ alerts
│     │ ├ rules
│     │ └ servicemonitors
│     └ services
├ environments # General defaults per environment
│ ├ dev
│ │ ├ components
│ │ │ ├ core
│ │ │ │   ├ configmaps
│ │ │ │   └ helmreleases
│ │ │ └ monitoring
│ │ │   ├ crds
│ │ │   │   ├ grafana-operator
│ │ │   │   │   └ crds
│ │ │   │   └ prometheus-operator
│ │ │   │       └ crds
│ │ │   └ helmreleases
│ │ ├ configmaps
│ │ ├ helmrepositories
│ │ └ podmonitors
│ ├ prd
│ │   ...
│ ├ ref
│ │   ...
│ └ tst
│     ...
├ locations # Cluster specific configs
│ ├ location-1
│ │ └ site-1
│ │   ├ customer_A-workload-cluster-1
│ │   │ ├ cni
│ │   │ │ ├ clusterrolebindings
│ │   │ │ ├ clusterroles
│ │   │ │ ├ configmaps
│ │   │ │ ├ crds
│ │   │ │ └ serviceaccounts
│ │   │ ├ configmaps
│ │   │ ├ gitrepositories
│ │   │ ├ kustomizations
│ │   │ └ secrets
│ │   └ customer_B-workload-cluster-1
│ │       ...
│ └ location-2
│     ...
├ network-zones
│ ├ environment-defaults # Contains the plain mainifest of each environment
│ │ ├ dev
│ │ │ ├ clusterrolebindings
│ │ │ ├ clusterroles
│ │ │ ├ crds
│ │ │ ├ networkpolicies
│ │ │ ├ serviceaccounts
│ │ │ └ services
│ │ ├ prd
│ │ │   ...
│ │ ├ ref
│ │ │   ...
│ │ └ tst
│ │     ...
│ ├ network-segment-1 # Specific config for network segment 1
│ │   ... # Contains the kustomize overlays used to modify the base manifests for each environment
│ └ network-segment-2
│     ...
└ providers # CAPI provider defaults and specific configs per environment
  ├ default
  ├ metal3
  │ ├ default
  │ │ ├ configmaps
  │ │ └ namespaces
  │ ├ dev
  │ │ └ helmreleases
  │ ├ prd
  │ │ └ helmreleases
  │ ├ ref
  │ │ ├ crds
  │ │ ├ helmreleases
  │ │ └ helmrepositories
  │ └ tst
  │   └ helmreleases
  └ vsphere
      ...

```