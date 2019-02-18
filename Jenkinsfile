pipeline {
    agent any

    stages {
        stage('Linting') {
            steps {

            }
        }
        stage('Deploying') {
            agent {
                dockerfile true
            }
            steps {
                sh './deploy.sh'
            }
        }
    }
}