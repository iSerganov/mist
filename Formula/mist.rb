class Mist < Formula
  desc "Asymmetric-key audio steganography for Go"
  homepage "https://github.com/iSerganov/mist"
  url "https://github.com/iSerganov/mist/archive/refs/tags/v1.2.0.tar.gz"
  sha256 "18181e6e4a7978f0cb9f2944650c784152c3020f5e0a274fc8cb7d0168453d56"
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
