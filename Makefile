DOCKER_NAME=autoupdate-server
DOCKER_IMAGE=getlantern/$(DOCKER_NAME)

PRIVATE_KEY_DIR?=/etc/private
WORKDIR?=workdir

DEPLOY_URL ?= deploy@update-stage.getlantern.org

RUN_MODE ?=

clean:
	rm -rf autoupdate-server patches assets workdir

docker:
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	mkdir -p $(WORKDIR) && \
	(docker stop $(DOCKER_NAME) || exit 0) && \
	(docker rm $(DOCKER_NAME) || exit 0) && \
	docker run -d  \
		-e RUN_MODE="$(RUN_MODE)" \
		-p 0.0.0.0:9999:9999 \
		--privileged \
		-v $(WORKDIR):/app \
		-v $(PRIVATE_KEY_DIR):/keys \
		--restart always \
		--memory-swappiness=0 \
		--name $(DOCKER_NAME) \
		$(DOCKER_IMAGE)

deploy: clean
	rsync -av --delete --exclude ".git" --exclude ".*.sw?" . $(DEPLOY_URL):~/deploy && \
	ssh $(DEPLOY_URL) 'cd ~/deploy && make docker && PRIVATE_KEY_DIR=~/private WORKDIR=~/tmp make docker-run'

mock-server: docker
	PRIVATE_KEY_DIR=$(PWD)/_resources/example-keys WORKDIR=/tmp RUN_MODE=mock $(MAKE) docker-run

mock-server-logs:
	docker logs --tail 20 -f $(DOCKER_NAME)

production:
	DEPLOY_URL=deploy@update.getlantern.org make deploy

stage:
	DEPLOY_URL=deploy@update-stage.getlantern.org make deploy
