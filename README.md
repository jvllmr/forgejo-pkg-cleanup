# forgejo-pkg-cleanup

An action to cleanup forgejo packages based on a set of rules.

```yaml
steps:
  - uses: https://codeberg.org/jvllmr/forgejo-pkg-cleanup@v1.0.0
    with:
      instance: https://codeberg.org
      username: jvllmr
      password: top-secret
      package: forgejo-pkg-cleanup
      packageType: container
      owner: jvllmr
      keepVersions: ^(?:v)?(?:\d+)(?:\.(\d+))?(?:\.(\d+))?(?:[a|b]\d+)?$
      retention: 2d
```

See [action.yml](./action.yml) for available inputs and outputs.
