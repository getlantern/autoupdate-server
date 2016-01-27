DOCKER_NAME=autoupdate-server
DOCKER_IMAGE=getlantern/$(DOCKER_NAME)

PRIVATE_KEY_DIR?=/etc/private
WORKDIR?=workdir

DEPLOY_URL ?= deploy@update-stage.getlantern.org

clean:
	rm -rf autoupdate-server patches assets workdir

docker:
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	mkdir -p $(WORKDIR) && \
	(docker stop $(DOCKER_NAME) || exit 0) && \
	(docker rm $(DOCKER_NAME) || exit 0) && \
	docker run -d  \
		-v $(WORKDIR):/app \
		-v $(PRIVATE_KEY_DIR):/keys \
		--restart always \
		--memory-swappiness=0 \
		--name $(DOCKER_NAME) \
		$(DOCKER_IMAGE)

deploy: clean
	rsync -a . $(DEPLOY_URL):~/deploy && \
	ssh $(DEPLOY_URL) 'cd ~/deploy && make docker && PRIVATE_KEY_DIR=~ WORKDIR=~/tmp make docker-run'

stage:
	DEPLOY_URL=deploy@update-stage.getlantern.org make deploy
