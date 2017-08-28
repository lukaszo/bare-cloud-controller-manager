FROM golang

COPY main /bare
COPY haproxy_template.cfg /haproxy_template.cfg

CMD ["/bare", "--kubeconfig", "/kube/config"]
