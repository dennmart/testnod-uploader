# TestNod Uploader - CI Integration

Each upload must include a `build-id` so TestNod can group parallel runners or matrix shards from the same CI build into one logical test run. Use whatever value uniquely identifies a build within your CI provider — the canonical choice for each provider is shown below.

If uploads are paused for your organization or project, every upload job fails with a `403` and the reason from TestNod in its log (or passes, if the integration sets `-ignore-failures`). A separate finalize step for the same build then reports that no test run was found for the build ID; the `403` in the upload jobs is the cause.

## GitHub Actions

```yaml
- name: Upload results to TestNod
  uses: testnod/testnod-uploader@v1
  with:
    token: ${{ secrets.TESTNOD_TOKEN }}
    file: results.xml
    build-id: ${{ github.run_id }}
```

## CircleCI

```yaml
orbs:
  testnod: testnod/uploader@1.0.0

workflows:
  test:
    jobs:
      - test
      - testnod/upload:
          token: TESTNOD_TOKEN
          file: results.xml
          build-id: ${CIRCLE_WORKFLOW_ID}
          requires:
            - test
```

## Jenkins

```groovy
pipeline {
    agent any
    stages {
        stage('Test') {
            steps {
                sh './run-tests.sh'
            }
        }
        stage('Upload') {
            steps {
                testnodUpload token: credentials('testnod-token'),
                              file: 'results.xml',
                              buildId: env.BUILD_TAG
            }
        }
    }
}
```

## GitLab CI

Use the [TestNod CI/CD component](https://gitlab.com/testnod/testnod-uploader). It reads the token from the `TESTNOD_TOKEN` CI/CD variable and uses `CI_PIPELINE_ID` as the build ID unless you set the `build-id` input.

```yaml
include:
  - component: gitlab.com/testnod/testnod-uploader/upload@2.1.0
    inputs:
      file: results.xml
```
