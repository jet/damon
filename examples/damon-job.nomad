job "damon-job" {
    region      = "eastus2"
    datacenters = ["dev"]
    type        = "service"
    group "damon-group" {
        count = 1
        task "damon-test-task" {
            artifact {
                ## Replace whith wheverer damon is hosted
                source = "https://example.com/damon.exe.zip"
            }
            artifact {
                ## Replace with wherever your service artifact is hosted
                source = "https://example.com/myservice.exe.zip"
            }
            driver = "raw_exec"
            config {
                # Damon is the task entrypoint
                command = "${NOMAD_TASK_DIR}/damon.exe"
                # Your command and services should be in the 'args' section
                args    = ["${NOMAD_TASK_DIR}/myservice.exe"]
            }
            resources {
                cpu    = 1024
                memory = 1024
                network {
                    mbits = 100
                    
                    ## For your own http endpoint
                    port "http" {}
                    
                    ## For metrics exposed by Damon
                    port "damon" {}
                }
            }
            env {
                "DAMON_ENFORCE_CPU_LIMIT"    = "Y
                "DAMON_ENFORCE_MEMORY_LIMIT" = "Y"
                
                ## Example of overriding NOMAD_CPU_LIMIT to give it more CPU than allocated
                "DAMON_CPU_LIMIT" = "2048"
            }
            ## For your own HTTP endpoint service
            service {
                port = "http"
                check {
                    type     = "http"
                    path     = "/health"
                    interval = "10s"
                    timeout  = "2s"   
                }
            }
            ## For damon's metrics endpoint
            service {
                port = "damon"
                name = "${NOMAD_TASK_NAME}-damon"
            }
        }
    }
}
