FROM opensuse/tumbleweed

RUN zypper in -y pcre2-devel go1.21 diffutils

COPY . /home/tux/dev
RUN git clone https://github.com/sekiguchi-nagisa/arsh.git
RUN zypper addrepo --no-gpgcheck -f https://download.opensuse.org/repositories/home:nsekiguchi/openSUSE_Tumbleweed/home:nsekiguchi.repo && \
    zypper refresh && \
    zypper install -y arsh

WORKDIR /home/tux/dev/
CMD ls ./ && go build -buildvcs=false -v ./... && \
    arsh ./test/run_all_tests.ds /arsh/tools/litecheck/litecheck
