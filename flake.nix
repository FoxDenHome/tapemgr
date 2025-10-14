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

        ltfs = pkgs.stdenv.mkDerivation rec {
          pname = "ltfs";
          version = "1.0.0";

          src = pkgs.fetchFromGitHub {
            rev = "9d232e551628d954c41497a45aedf2315ffca591";
            owner = "LinearTapeFileSystem";
            repo = "ltfs";
            sha256 = "193593hsc8nf5dn1fkxhzs1z4fpjh64hdkc8q6n9fgplrpxdlr4s";
          };

          sourceRoot = "${src.name}/ltfs";

          # include sys/sysctl.h is deprecated in glibc. The sysctl calls are only used
          # for Apple to determine the kernel version. Because this build only targets
          # Linux is it safe to remove.
          patches = [ ./remove-sysctl.patch ];

          nativeBuildInputs = [ pkgs.pkg-config ];

          buildInputs = with pkgs; [
            fuse
            icu66
            libxml2
            libuuid
          ];
        };
      in
      {
        packages = {
          inherit tapemgr ltfs;
          default = tapemgr;
        };
      });
}
