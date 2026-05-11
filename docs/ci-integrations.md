# TestNod Uploader - CI Integration

Each upload must include a `build-id` so TestNod can group parallel runners or matrix shards from the same CI build into one logical test run. Use whatever value uniquely identifies a build within your CI provider — the canonical choice for each provider is shown below.

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

```yaml
include:
  - remote: 'https://raw.githubusercontent.com/testnod/testnod-uploader/main/gitlab/testnod.yml'

upload_results:
  extends: .testnod-upload
  variables:
    TESTNOD_TOKEN: $TESTNOD_TOKEN
    TESTNOD_FILE: results.xml
    TESTNOD_BUILD_ID: $CI_PIPELINE_ID
```
