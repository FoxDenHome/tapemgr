{
  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }: 
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        tapemgr = pkgs.buildGoModule {
          pname = "tapemgr";
          version = "1.0.0";
          src = ./.;
          vendorHash = "sha256-yLG9GbvJ/U8CS2rJgIy2aP4+Oj386HQ6TpmlGWVqrpg=";
          buildInputs = [];
        };

        tapemgr-ltfs = pkgs.stdenv.mkDerivation rec {
          pname = "tapemgr-ltfs";
          version = "1.0.0";

          src = pkgs.fetchFromGitHub {
            rev = "9d232e551628d954c41497a45aedf2315ffca591";
            owner = "LinearTapeFileSystem";
            repo = "ltfs";
            hash = "sha256-lUEE47EblyoF53OktDTPHbNQ514nTLPMPSbciiLESGA=";
          };

          sourceRoot = "${src.name}";

          nativeBuildInputs = with pkgs; [
            pkg-config
            autoreconfHook
          ];

          buildInputs = with pkgs; [
            fuse
            icu66
            libxml2
            libuuid
          ];

          configureFlags = [
            "--disable-snmp"
          ];
        };
      in
      {
        packages = {
          inherit tapemgr tapemgr-ltfs;
          default = tapemgr;
        };
      });
}
