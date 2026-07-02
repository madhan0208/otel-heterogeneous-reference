pipeline {
    agent any

    stages {
        stage('Install dependencies') {
            steps {
                sh 'python3 -m pip install pytest requests opentelemetry-proto'
            }
        }

        stage('Run MVS compliance tests') {
            steps {
                sh 'python3 -m pytest tests/mvs_compliance.py -v'
            }
        }
    }

    post {
        success {
            echo 'Pipeline passed — MVS compliance verified'
        }
        failure {
            echo 'Pipeline failed — MVS compliance violation detected'
        }
    }
}
