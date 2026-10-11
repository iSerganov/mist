class Mist < Formula
  desc "Asymmetric-key audio steganography for Go"
  homepage "https://github.com/iSerganov/mist"
  url "https://github.com/iSerganov/mist/archive/refs/tags/v1.1.0.tar.gz"
  sha256 "7ca0452b9793d19ffabac06ce89d50a5082d6ededdffe575a9af0894cf0427a7"
  license "MIT"
  head "https://github.com/iSerganov/mist.git", branch: "master"

  depends_on "go" => :build
  depends_on "pkgconf" => :build
  depends_on "ffmpeg"

  def install
    ENV["CGO_ENABLED"] = "1"
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/mist"
  end

  test do
    assert_match "ogg", shell_output("#{bin}/mist formats")
  end
end
