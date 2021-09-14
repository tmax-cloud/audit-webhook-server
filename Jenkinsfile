node {
	def hcIntegratedBuildDir = "/var/lib/jenkins/workspace/audit-webhook-server"

	DisAuditWebhookServer()

	stage('clean repo') {
        sh "sudo rm -rf ${hcIntegratedBuildDir}/*"
    }
}

void DisAuditWebhookServer() {
    def gitHubBaseAddress = "github.com"
    def gitAddress = "${gitHubBaseAddress}/tmax-cloud/audit-webhook-server.git"
    def homeDir = "/var/lib/jenkins/workspace/audit-webhook-server"
    def scriptHome = "${homeDir}/scripts"
    def version = "${params.majorVersion}.${params.minorVersion}.${params.tinyVersion}.${params.hotfixVersion}"
    def imageTag = "b${version}"
    def userName = "aldlfkahs"
    def userEmail = "seungwon_lee@tmax.co.kr"
    def githubUserToken = "${params.githubUserToken}"

    dir(homeDir){
        stage('Audit-Webhook-Server (git pull)') {
            git branch: "master",
            credentialsId: '${userName}',
            url: "http://${gitAddress}"

            // git pull
            sh "git checkout master"
            sh "git config --global user.name ${userName}"
            sh "git config --global user.email ${userEmail}"
            sh "git config --global credential.helper store"

            sh "git fetch --all"
            sh "git reset --hard origin/master"
            sh "git pull origin master"
        }

        stage('Audit-Webhook-Server (image build & push)'){
            sh "sudo docker build --tag tmaxcloudck/audit-webhook-server:${imageTag} ."
            sh "sudo docker push tmaxcloudck/audit-webhook-server:${imageTag}"
            sh "sudo docker rmi tmaxcloudck/audit-webhook-server:${imageTag}"
        }

        stage('Audit-Webhook-Server (make change log)'){
            preVersion = sh(script:"sudo git describe --tags --abbrev=0", returnStdout: true)
            preVersion = preVersion.substring(1)
            echo "preVersion of audit-webhook-server : ${preVersion}"
            sh "sudo sh ${scriptHome}/audit-webhook-server-changelog.sh ${version} ${preVersion}"
        }

        stage('Audit-Webhook-Server (git push)'){
            sh "git checkout master"

            sh "git config --global user.name ${userName}"
            sh "git config --global user.email ${userEmail}"
            sh "git config --global credential.helper store"
            sh "git add -A"

            def commitMsg = "[Distribution] Release commit for Audit-Webhook-Server-v${version}"
            sh (script: "git commit -m \"${commitMsg}\" || true")
            sh "git tag v${version}"

            sh "git remote set-url origin https://${githubUserToken}@github.com/tmax-cloud/audit-webhook-server.git"
            sh "sudo git push -u origin +master"
            sh "sudo git push origin v${version}"
        }

        stage('Audit-Webhook-Server (gh release upload)'){
            sh "gh auth login -w \"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\""
            sh "gh release create v${version} -t v${version} -n \"Release v${version}\""
            sh "gh release upload v5.0.1.0 install-yaml/01_timescaledb.yaml"
            sh "gh release upload v5.0.1.0 install-yaml/02_audit-deployment.yaml"

            sh "git fetch --all"
            sh "git reset --hard origin/master"
            sh "git pull origin master"
        }
    }
}