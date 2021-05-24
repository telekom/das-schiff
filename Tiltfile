load('ext://restart_process', 'docker_build_with_restart')

def kubebuilder(DOMAIN, IMG='controller:latest', CONTROLLERGEN='crd:trivialVersions=true rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases;'):

    DOCKERFILE = '''FROM golang:alpine
    WORKDIR /
    COPY ./bin/manager /
    CMD ["/manager"]
    '''

    def yaml():
        return local('cd config/manager; kustomize edit set image controller=' + IMG + '; cd ../..; kustomize build config/default')
    
    def manifests():
        return 'bin/controller-gen ' + CONTROLLERGEN
    
    def generate():
        return 'bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./...";'
    
    def vetfmt():
        return 'go vet ./...; go fmt ./...'
    
    def binary():
        return 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o bin/manager main.go'

    installed = local("which kubebuilder")
    print("kubebuilder is present:", installed)

    DIRNAME = os.path.basename(os. getcwd())
    # if kubebuilder
    if os.path.exists('go.mod') == False:
        local("go mod init %s" % DIRNAME)
    
    if os.path.exists('PROJECT') == False:
        local("kubebuilder init --domain %s" % DOMAIN)
    
    # if os.path.exists('api') == False:
    #     local("kubebuilder create api --resource --controller --group %s --version %s --kind %s" % (GROUP, VERSION, KIND))
    
    local(manifests() + generate())
    
    #local_resource('CRD', manifests() + 'kustomize build config/crd | kubectl apply -f -', deps=["api"])
    
    k8s_yaml(yaml())
    
    deps = ['controllers', 'main.go']
    deps.append('api')
    
    local_resource('Watch&Compile', generate() + binary(), deps=deps, ignore=['*/*/zz_generated.deepcopy.go'])
    
    local_resource('Sample YAML', 'kubectl apply -f ./config/samples/', deps=["./config/samples"], resource_deps=[DIRNAME + "-controller-manager"])
    local_resource('CAPI CRDs', 'kubectl apply -f ./config/crd/testing', deps=["./config/crd/testing"], resource_deps=[DIRNAME + "-controller-manager"])

    docker_build_with_restart(IMG, '.', 
     dockerfile_contents=DOCKERFILE,
     entrypoint='/manager',
     only=['./bin/manager'],
     live_update=[
           sync('./bin/manager', '/manager'),
       ]
    )
kubebuilder("schiff.telekom.de") 