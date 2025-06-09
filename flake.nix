{
  description = "opa";
  inputs = {
    nixpkgs = {
      type = "github";
      owner = "NixOS";
      repo = "nixpkgs";
      rev = "70c74b02eac46f4e4aa071e45a6189ce0f6d9265";
    };
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    { nixpkgs, flake-utils, ... }:
    let
      utils = flake-utils;
    in
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config = {
            allowUnfree = true;
          };
        };
      in
      {
        formatter = pkgs.nixpkgs-fmt;
        devShell = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            go_1_23
            claude-code
          ];
          shellHook = '''';
        };
      }
    );
}
