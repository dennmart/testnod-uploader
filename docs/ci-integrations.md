# TestNod Uploader - CI Integration

## GitHub Actions

```yaml
- name: Upload results to TestNod
  uses: testnod/testnod-uploader@v1
  with:
    token: ${{ secrets.TESTNOD_TOKEN }}
    file: results.xml
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
                              file: 'results.xml'
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
```
