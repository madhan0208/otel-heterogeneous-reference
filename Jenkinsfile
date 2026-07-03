pipeline 
{
    agent 
    {
        docker {
            image 'python:3.12-slim'
            args '-v /var/run/docker.sock:/var/run/docker.sock'
        }
    }

    stages 
    {
        stage('Install dependencies') {
            steps {
                sh 'pip install --target=/tmp/pip-packages pytest requests opentelemetry-proto'
            }
        }

        stage('Run MVS compliance tests') {
            steps {
                sh 'PYTHONPATH=/tmp/pip-packages python3 -m pytest tests/mvs_compliance.py -v'
            }
        }
    }

    post {
        success 
        {
            echo 'Pipeline passed — MVS compliance verified'
        }
        failure 
        {
            echo 'Pipeline failed — MVS compliance violation detected'
        }
    }
}
