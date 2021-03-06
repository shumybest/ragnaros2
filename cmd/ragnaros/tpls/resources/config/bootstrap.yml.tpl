# ===================================================================
# Spring Cloud Config bootstrap configuration for the "dev" profile
# In prod profile, properties will be overwritten by the ones defined in bootstrap-prod.yml
# ===================================================================

jhipster:
  registry:
    password: registryPasswordHere

registry-host: ${REGISTRY_IP:127.0.0.1}

spring:
  application:
    name: {{ .App.ProjectName }}
  profiles:
    # The commented value for `active` can be replaced with valid Spring profiles to load.
    # Otherwise, it will be filled in by maven when building the WAR file
    # Either way, it can be overridden by `--spring.profiles.active` value passed in the commandline or `-Dspring.profiles.active` set in `JAVA_OPTS`
    active: {{ .K8s.Spring.Profiles }}
  cloud:
    config:
      fail-fast: false # if not in "prod" profile, do not force to use Spring Cloud Config
      uri: http://admin:${jhipster.registry.password}@${registry-host}:8761/config
      # name of the config server's property source (file.yml) that we want to use
      name: {{ .App.ProjectName }}
      profile: {{ .K8s.Spring.Profiles }} # profile(s) of the property source
      label: master # toggle to switch to a different version of the configuration as stored in git
      # it can be set to any label, branch or commit of the configuration source Git repository
