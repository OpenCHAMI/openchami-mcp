{
  description = "Container and package builds for openchami-mcp";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    {
      self,
      flake-utils,
      nixpkgs,
    }:
    flake-utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
    ] (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };

        lib = pkgs.lib;
        version = if self ? shortRev then self.shortRev else "dirty";

        openchamiMcp = pkgs.buildGoModule {
          pname = "openchami-mcp";
          inherit version;
          src = self;
          vendorHash = null;
          subPackages = [ "cmd/openchami-mcp" ];
          ldflags = [
            "-s"
            "-w"
          ];
          env.CGO_ENABLED = 0;
          doCheck = true;
          checkPhase = ''
            runHook preCheck
            go test ./...
            runHook postCheck
          '';
        };

        imageRoot = pkgs.buildEnv {
          name = "openchami-mcp-image-root";
          paths = [
            openchamiMcp
            pkgs.tini
            pkgs.cacert
          ];
          pathsToLink = [
            "/bin"
            "/etc/ssl/certs"
          ];
        };

        defaultEnv = [
          "OPENCHAMI_MCP_MODE=read-only"
          "OPENCHAMI_MCP_ENABLE_WRITES=false"
          "OPENCHAMI_MCP_REQUIRE_WRITE_ACK=true"
          "OPENCHAMI_MCP_FRAMING=auto"
          "OPENCHAMI_MCP_LOG_LEVEL=info"
          "OPENCHAMI_MCP_SMD_TIMEOUT=10s"
          "OPENCHAMI_MCP_SMD_RETRIES=1"
          "OPENCHAMI_MCP_BSS_TIMEOUT=10s"
          "OPENCHAMI_MCP_BSS_RETRIES=1"
          "OPENCHAMI_MCP_CLOUD_INIT_TIMEOUT=10s"
          "OPENCHAMI_MCP_CLOUD_INIT_RETRIES=1"
          "OPENCHAMI_MCP_POWER_CONTROL_TIMEOUT=10s"
          "OPENCHAMI_MCP_POWER_CONTROL_RETRIES=1"
          "OPENCHAMI_MCP_KEA_TIMEOUT=10s"
          "OPENCHAMI_MCP_KEA_RETRIES=1"
          "OPENCHAMI_MCP_KEA_DEFAULT_SERVICES=dhcp4"
          "OPENCHAMI_MCP_COREDHCP_TIMEOUT=10s"
          "OPENCHAMI_MCP_COREDHCP_RETRIES=1"
          "SSL_CERT_FILE=${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt"
        ];

        skopeoPolicy = pkgs.writeText "containers-policy.json" ''
          {
            "default": [
              {
                "type": "insecureAcceptAnything"
              }
            ]
          }
        '';

        dockerImage = pkgs.dockerTools.buildImage {
          name = "openchami-mcp";
          tag = version;
          copyToRoot = imageRoot;
          extraCommands = ''
            mkdir -p tmp
            chmod 1777 tmp
          '';
          config = {
            Entrypoint = [
              "/bin/tini"
              "--"
            ];
            Cmd = [ "/bin/openchami-mcp" ];
            Env = defaultEnv;
            User = "65534:65534";
            WorkingDir = "/";
          };
        };

        ociImage = pkgs.runCommand "openchami-mcp-oci-image-${version}.tar" { nativeBuildInputs = [ pkgs.skopeo ]; } ''
          tmpdir="$PWD/tmp"
          mkdir -p "$tmpdir"
          skopeo --policy ${skopeoPolicy} copy --tmpdir "$tmpdir" docker-archive:${dockerImage} oci-archive:$out:openchami-mcp:${version}
        '';
      in
      {
        packages = {
          default = openchamiMcp;
          dockerImage = dockerImage;
          ociImage = ociImage;
        };

        apps.default = flake-utils.lib.mkApp {
          drv = openchamiMcp;
          exePath = "/bin/openchami-mcp";
        };

        checks = {
          default = openchamiMcp;
          ociImage = ociImage;
        };

        formatter = pkgs.nixfmt-tree;
      }
    );
}
