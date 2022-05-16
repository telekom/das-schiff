# Das Schiff Liquid Metal

In order to scale Das Schiff to edge locations, we were looking for a solution that allows to efficiently use the limited resources available at such locations. Using entire bare metal machines as nodes, especially for the control plane, is not really an option if only a few servers are available. At the same time, some workloads require high performance, or even direct access to hardware resources.

Together with our friends from Weaveworks we started looking at a few existing solutions like K3s, K0s, MicroK8s or using mixed node clusters, but none of them suited were satisfying all of our goals. We then started thinking about using microVMs on those bare metal machines, and Weaveworks liked the idea so much that they started building Weaveworks Liquid Metal. 

You can find out all about at their GitHub Org: https://github.com/weaveworks-liquidmetal