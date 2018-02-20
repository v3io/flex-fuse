FROM alpine
ADD v3fs-deploy.sh /
CMD ["/bin/ash","v3fs-deploy.sh"]
