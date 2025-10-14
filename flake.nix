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
      in
      {
        packages = {
          inherit tapemgr;
          default = tapemgr;
        };
      });
}
