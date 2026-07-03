pipeline {
    agent any

    // --- SETUP TOOLS ---
    tools {
        // Pastikan nama ini sesuai dengan "Global Tool Configuration" di Jenkins
        go 'Go-1.23' 
    }

    environment {
        // --- CONFIG ---
        DOCKER_USER  = "dockerdevopsethos"
        APP_NAME     = "be-extension-erp-api"
        IMAGE_TAG    = "${DOCKER_USER}/${APP_NAME}:${BUILD_NUMBER}"
        LATEST_TAG   = "${DOCKER_USER}/${APP_NAME}:latest"
        
        // --- SERVER TUJUAN ---
        DEPLOY_USER  = "root"
        DEPLOY_HOST  = "89.21.85.2" 
        DEPLOY_DIR   = "/var/www/html/be-extension-erp"
        
        // --- CREDENTIALS ID ---
        DOCKER_CREDS = credentials('docker-hub-login')
        ENV_SECRET   = credentials('ENV-BE-HM-DEV') // File .env rahasia
        SSH_CREDS_ID = 'ssh-server-deploy' // SSH Key
    }

    stages {
        stage('1. Checkout') {
            steps {
                cleanWs()
                checkout([
                    $class: 'GitSCM', 
                    branches: [[name: '*/development']], // Sesuaikan dengan branch yang ingin Anda build
                    userRemoteConfigs: [[
                        url: 'git@github.com:Extension-ERP/be-extension-erp.git', // Sesuaikan dengan repository Anda
                        credentialsId: 'ssh-server-deploy'
                    ]]
                ])
            }
        }

        stage('2. Test & Coverage') {
            steps {
                sh 'go version' 
                sh 'go mod download'
                sh 'go test -json > test-report.out || true'
                sh 'go test -coverprofile=coverage.out ./... || true'
            }
        }

        stage('3. SonarQube Analysis') {
            steps {
                script {
                    def scannerHome = tool 'SonarScanner' 
                    withSonarQubeEnv('SONAR-BE-EXTENSION-DEV') { 
                        sh "${scannerHome}/bin/sonar-scanner"
                    }
                }
            }
        }

        stage('4. Quality Gate') {
            steps {
                script {
                    timeout(time: 2, unit: 'MINUTES') {
                        waitForQualityGate abortPipeline: true
                    }
                }
            }
        }

        stage('5. Build & Push Docker') {
            steps {
                script {
                    sh "docker build -t ${IMAGE_TAG} ."
                    sh "docker tag ${IMAGE_TAG} ${LATEST_TAG}"
                    
                    withCredentials([usernamePassword(credentialsId: 'docker-hub-login', passwordVariable: 'PASS', usernameVariable: 'USER')]) {
                        sh "echo $PASS | docker login -u $USER --password-stdin"
                        sh "docker push ${IMAGE_TAG}"
                        sh "docker push ${LATEST_TAG}"
                    }
                }
            }
        }

        stage('6. Deploy Production (SSH)') {
            steps {
                sshagent([SSH_CREDS_ID]) {
                    script {
                        // ---------------------------------------------------------
                        // LANGKAH 1: PERSIAPAN FILE .ENV
                        // ---------------------------------------------------------
                        
                        // A. BACA isi file rahasia dari Jenkins (Bukan path-nya)
                        def secretContent = readFile(file: ENV_SECRET)
                        
                        // B. Gabungkan isi rahasia + Variable Image Tag
                        def finalEnvContent = "${secretContent}\nFULL_IMAGE_NAME=${LATEST_TAG}"

                        // C. Tulis menjadi file .env fisik di Workspace Jenkins
                        writeFile file: '.env', text: finalEnvContent

                        // Debug: Pastikan formatnya KEY=VALUE (Bukan /var/jenkins/...)
                        echo "--- PREVIEW 3 BARIS PERTAMA .ENV ---"
                        sh "head -n 3 .env"

                        // ---------------------------------------------------------
                        // LANGKAH 2: KIRIM FILE KE SERVER
                        // ---------------------------------------------------------
                        // Kita kirim .env yang SUDAH DIPERBAIKI di atas ke server
                        sh "scp -o StrictHostKeyChecking=no docker-compose.yml .env ${DEPLOY_USER}@${DEPLOY_HOST}:${DEPLOY_DIR}/"
                        
                        // ---------------------------------------------------------
                        // LANGKAH 3: EKSEKUSI DOCKER DI SERVER
                        // ---------------------------------------------------------
                        sh """
                            ssh -o StrictHostKeyChecking=no ${DEPLOY_USER}@${DEPLOY_HOST} '
                                cd ${DEPLOY_DIR}
                                echo "🚀 Connected to Server..."
                                
                                # Stop dan hapus semua container yang terkait
                                docker stop be-extension-erp-api be-extension-erp-redis 2>/dev/null || true
                                docker rm be-extension-erp-api be-extension-erp-redis 2>/dev/null || true
                                
                                # Matikan via compose (TANPA --volumes untuk preserve data)
                                docker compose down --remove-orphans
                                
                                # Pull image baru (Docker akan baca FULL_IMAGE_NAME dari .env)
                                docker compose pull
                                
                                # Nyalakan container dengan force recreate
                                docker compose up -d --force-recreate
                                
                                # Tunggu container stabil
                                sleep 5
                                
                                # Cek status container dan port yang published
                                echo "📋 Container Status:"
                                docker compose ps
                                echo ""
                                echo "� Verify Published Ports:"
                                docker ps --filter name=be-extension-erp --format "table {{.Names}}\t{{.Ports}}"
                                echo ""
                                echo "📝 Container Logs (last 100 lines):"
                                docker compose logs --tail=100 api 2>&1 | head -100
                                echo ""
                                echo "🔎 Container Inspect (State):"
                                docker inspect be-extension-erp-api --format "{{.State.Status}} | Running: {{.State.Running}} | Restarting: {{.State.Restarting}} | ExitCode: {{.State.ExitCode}}"
                                echo ""
                                echo "🧪 Health Check from Server (verbose):"
                                curl -v http://localhost:9192/health 2>&1 || echo "❌ Health check failed"
                                
                                # Bersihkan image lama
                                docker image prune -f
                                
                                echo "✅ Deployment Selesai!"
                            '
                        """
                    }
                }
            }
        }
    }
        
    post {
        always {
            script {
                // Bersihkan image sampah di Jenkins (Hemat Disk Space)
                sh "docker rmi ${IMAGE_TAG} || true"
                sh "docker image prune -f"
            }
            cleanWs()
        }
        success {
            echo "✅ Deployment Sukses di Server Host: ${DEPLOY_DIR}"
        }
        failure {
            echo "❌ Deployment Gagal."
        }
    }
}
