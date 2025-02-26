FROM almalinux:9 as base

# make build args available as ENV variables to downstream images
# so that we don't have to pass the same build args again
ARG APP_USER_ID=1000
ARG APP_GROUP_ID=1000
ARG APP_USER_NAME=chrome-headless
ARG APP_GROUP_NAME=chrome-headless

ARG APP_DNF_PACKAGES="chromium-headless socat"
ARG APP_DNF_REPOS=""

#chromium-headless socat mscorefonts php-pdf-fonts
ENV APP_USER_ID=${APP_USER_ID}
ENV APP_GROUP_ID=${APP_GROUP_ID}
ENV APP_USER_NAME=${APP_USER_NAME}
ENV APP_GROUP_NAME=${APP_GROUP_NAME}

# Install Packages
RUN for repo in ${APP_DNF_REPOS}; do curl $repo >> /etc/yum.repos.d/additional-repos.repo; done && \
    dnf install -y epel-release && \
    dnf config-manager --set-enabled crb && \
    dnf install -y ${APP_DNF_PACKAGES} && \
    dnf clean all

RUN groupadd -g $APP_GROUP_ID $APP_GROUP_NAME && \
    adduser -u $APP_USER_ID -s /bin/bash -g $APP_GROUP_NAME $APP_USER_NAME

USER ${APP_USER_NAME}

COPY ./run.sh /usr/local/bin/

EXPOSE 1337

CMD /usr/local/bin/run.sh
