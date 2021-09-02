FROM registry.access.redhat.com/ubi8/ubi:8.4

RUN yum -y install unzip wget

RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
RUN unzip awscliv2.zip
RUN ./aws/install

RUN wget https://dl.google.com/go/go1.15.5.linux-amd64.tar.gz

RUN tar -C /usr/local -xf go1.15.5.linux-amd64.tar.gz

RUN set -o pipefail && \
    curl -Ls https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/linux/oc.tar.gz | \
    tar -zx && \
    mv oc /usr/local/bin

ENV PATH="${PATH}:/usr/local/go/bin"

COPY . /usr/local/bin/

RUN chmod a+rwx -R /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/backup.sh"]
