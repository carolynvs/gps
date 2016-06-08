package vsolver

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// A remoteRepo represents a potential remote repository resource.
//
// RemoteRepos are based purely on lexical analysis; successfully constructing
// one is not a guarantee that the resource it identifies actually exists or is
// accessible.
type remoteRepo struct {
	Base     string
	RelPkg   string
	CloneURL *url.URL
	Schemes  []string
	VCS      []string
}

//type remoteResult struct {
//r   remoteRepo
//err error
//}

// TODO sync access to this map
//var remoteCache = make(map[string]remoteResult)

// Regexes for the different known import path flavors
var (
	ghRegex      = regexp.MustCompile(`^(?P<root>github\.com/([A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+))(/[A-Za-z0-9_.\-]+)*$`)
	gpinNewRegex = regexp.MustCompile(`^(?P<root>gopkg\.in/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(-unstable)?)(?:\.git))?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)
	//gpinOldRegex = regexp.MustCompile(`^(?P<root>gopkg\.in/(?:([a-z0-9][-a-z0-9]+)/)?((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(-unstable)?))/([a-zA-Z][-a-zA-Z0-9]*)(?:\.git)?((?:/[a-zA-Z][-a-zA-Z0-9]*)*)$`)
	bbRegex = regexp.MustCompile(`^(?P<root>bitbucket\.org/(?P<bitname>[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+))(/[A-Za-z0-9_.\-]+)*$`)
	lpRegex = regexp.MustCompile(`^(?P<root>launchpad.net/([A-Za-z0-9-._]+)(/[A-Za-z0-9-._]+)?)(/.+)?`)
	//glpRegex = regexp.MustCompile(`^(?P<root>git\.launchpad\.net/(([A-Za-z0-9_.\-]+)|~[A-Za-z0-9_.\-]+/(\+git|[A-Za-z0-9_.\-]+)/[A-Za-z0-9_.\-]+))$`)
	//gcRegex      = regexp.MustCompile(`^(?P<root>code\.google\.com/[pr]/(?P<project>[a-z0-9\-]+)(\.(?P<subrepo>[a-z0-9\-]+))?)(/[A-Za-z0-9_.\-]+)*$`)
	jazzRegex    = regexp.MustCompile(`^(?P<root>hub\.jazz\.net/git/[a-z0-9]+/[A-Za-z0-9_.\-]+)(/[A-Za-z0-9_.\-]+)*$`)
	genericRegex = regexp.MustCompile(`^(?P<root>(?P<repo>([a-z0-9.\-]+\.)+[a-z0-9.\-]+(:[0-9]+)?/[A-Za-z0-9_.\-/~]*?)\.(?P<vcs>bzr|git|hg|svn))([/A-Za-z0-9_.\-]+)*$`)
)

// Other helper regexes
var (
	scpSyntaxRe = regexp.MustCompile(`^([a-zA-Z0-9_]+)@([a-zA-Z0-9._-]+):(.*)$`)
	pathvld     = regexp.MustCompile(`^([A-Za-z0-9-]+)(\.[A-Za-z0-9-]+)+(/[A-Za-z0-9-_.~]+)*$`)
)

// deduceRemoteRepo takes a potential import path and returns a RemoteRepo
// representing the remote location of the source of an import path. Remote
// repositories can be bare import paths, or urls including a checkout scheme.
func deduceRemoteRepo(path string) (rr remoteRepo, err error) {
	if m := scpSyntaxRe.FindStringSubmatch(path); m != nil {
		// Match SCP-like syntax and convert it to a URL.
		// Eg, "git@github.com:user/repo" becomes
		// "ssh://git@github.com/user/repo".
		rr.CloneURL = &url.URL{
			Scheme:  "ssh",
			User:    url.User(m[1]),
			Host:    m[2],
			RawPath: m[3],
		}
	} else {
		rr.CloneURL, err = url.Parse(path)
		if err != nil {
			return nil, "", fmt.Errorf("%q is not a valid import path", path)
		}
	}

	path = rr.CloneURL.Host + rr.CloneURL.Path
	if !pathvld.MatchString(path) {
		return remoteRepo{}, fmt.Errorf("%q is not a valid import path", path)
	}

	if u.Scheme != "" {
		rr.Schemes = []string{u.Scheme}
	}

	switch {
	case ghRegex.MatchString(path):
		v := ghRegex.FindStringSubmatch(path)

		rr.CloneURL.Host = "github.com"
		rr.CloneURL.Path = v[2]
		rr.Base = v[1]
		rr.RelPkg = strings.TrimPrefix(v[3], "/")
		rr.VCS = []string{"git"}

		return

	case gpinNewRegex.MatchString(path):
		v := gpinNewRegex.FindStringSubmatch(path)

		// Duplicate some logic from the gopkg.in server in order to validate
		// the import path string without having to hit the server
		if strings.Contains(v[4], ".") {
			return remoteRepo{}, fmt.Errorf("%q is not a valid import path; gopkg.in only allows major versions (%q instead of %q)",
				path, v[4][:strings.Index(v[4], ".")], v[4])
		}

		// If the third position is empty, it's the shortened form that expands
		// to the go-pkg github user
		if v[3] != "" {
			rr.CloneURL.Path = "go-pkg/" + v[4]
		} else {
			rr.CloneURL.Path = v[2] + v[4]
		}
		rr.CloneURL.Host = "github.com"
		rr.Base = v[1]
		rr.RelPkg = strings.TrimPrefix(v[6], "/")
		rr.VCS = []string{"git"}

		return
	//case gpinOldRegex.MatchString(path):

	case bbRegex.MatchString(path):
		v := bbRegex.FindStringSubmatch(path)

		rr.CloneURL.Host = "bitbucket.org"
		rr.CloneURL.Path = v[2]
		rr.Base = v[1]
		rr.RelPkg = strings.TrimPrefix(v[5], "/")
		rr.VCS = []string{"git", "hg"}

		return

	//case gcRegex.MatchString(path):
	//v := gcRegex.FindStringSubmatch(path)

	//rr.CloneURL.Host = "code.google.com"
	//rr.CloneURL.Path = "p/" + v[2]
	//rr.Base = v[1]
	//rr.RelPkg = strings.TrimPrefix(v[5], "/")
	//rr.VCS = []string{"hg", "git"}

	//return

	case lpRegex.MatchString(path):
		v := lpRegex.FindStringSubmatch(path)
		v = append(v, "", "")

		rr.CloneURL.Host = "launchpad.net"
		rr.Base = v[1]
		rr.RelPkg = strings.TrimPrefix(v[4], "/")
		rr.VCS = []string{"bzr"}

		if v[3] == "" {
			// launchpad.net/project"
			rr.Base = fmt.Sprintf("https://launchpad.net/%v", v[2])
		} else {
			// launchpad.net/project/series"
			rr.Base = fmt.Sprintf("https://launchpad.net/%s/%s", v[2], v[3])
		}
		return

	//case glpRegex.MatchString(path):
	//// TODO too many rules for this, commenting out for now
	//v := lpRegex.FindStringSubmatch(path)

	//rr.CloneURL.Host = "launchpad.net"
	//rr.RelPkg = strings.TrimPrefix(v[3], "/")
	//rr.VCS = []string{"git"}

	//v = append(v, "", "")
	//if v[2] == "" {
	//// launchpad.net/project"
	//rr.Base = fmt.Sprintf("https://launchpad.net/%v", v[1])
	//} else {
	//// launchpad.net/project/series"
	//rr.Base = fmt.Sprintf("https://launchpad.net/%s/%s", v[1], v[2])
	//}
	//return

	// try the general syntax
	case genericRegex.MatchString(path):
		v := genericRegex.FindStringSubmatch(path)
		switch v[5] {
		case "git":
			x := strings.SplitN(v[1], "/", 2)
			rr.CloneURL.Host = x[0]
			rr.CloneURL.Path = x[1]
			rr.VCS = []string{"git"}
		case "hg":
			x := strings.SplitN(v[1], "/", 2)
			rr.CloneURL.Host = x[0]
			rr.CloneURL.Path = x[1]
			rr.VCS = []string{"hg"}
		case "bzr":
			repo, err := Bzrrepo("https://" + v[1])
			rr.VCS = []string{"bzr"}
		default:
			return remoteRepo{}, fmt.Errorf("unknown repository type: %q", v[5])

		}
		rr.RelPkg = strings.TrimPrefix(v[6], "/")
		return
	}

	// TODO use HTTP metadata to resolve vanity imports
	return remoteRepo{}, fmt.Errorf("unable to deduct repository and source type for: %q", path)
}
