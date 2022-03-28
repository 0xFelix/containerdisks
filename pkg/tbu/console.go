package tbu

import (
	"fmt"
	"regexp"
	"time"

	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"
	kvirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests/console"
)

// To be upstreamed
// Additions for kubevirt.io/kubevirt/tests/console

type LoginOptions struct {
	Username              string
	Password              string
	DoNotLoginRegexp      string
	LoginSuccessfulRegexp string
}

func NewLoginOptions(username, password, vmiName string) *LoginOptions {
	return &LoginOptions{
		Username:              username,
		Password:              password,
		DoNotLoginRegexp:      fmt.Sprintf(`(\[%s@(localhost|%s) ~\]\$ |\[root@(localhost|%s) %s\]\# )`, username, vmiName, vmiName, username),
		LoginSuccessfulRegexp: fmt.Sprintf(`\[%s@(localhost|%s) ~\]\$ `, username, vmiName),
	}
}

func LoginToGeneric(vmi *kvirtv1.VirtualMachineInstance, options *LoginOptions) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	expecter, _, err := console.NewExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	err = expecter.Send("\n")
	if err != nil {
		return err
	}

	// Do not login, if we already logged in
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: options.DoNotLoginRegexp},
	})
	_, err = expecter.ExpectBatch(b, 5*time.Second)
	if err == nil {
		return nil
	}

	b = append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				// and in case the VM's did not get hostname form DHCP server try the default hostname
				R:  regexp.MustCompile(fmt.Sprintf(`(localhost|%s) login: `, vmi.Name)),
				S:  fmt.Sprintf("%s\n", options.Username),
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Password:`),
				S:  fmt.Sprintf("%s\n", options.Password),
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Login incorrect`),
				T:  expect.LogContinue("Failed to log in", expect.NewStatus(codes.PermissionDenied, "login failed")),
				Rt: 10,
			},
			&expect.Case{
				R: regexp.MustCompile(options.LoginSuccessfulRegexp),
				T: expect.OK(),
			},
		}},
		&expect.BSnd{S: "sudo su\n"},
		&expect.BExp{R: console.PromptExpression},
	})
	res, err := expecter.ExpectBatch(b, 2*time.Minute)
	if err != nil {
		log.DefaultLogger().Object(vmi).Reason(err).Errorf("Login attempt failed: %+v", res)
		// Try once more since sometimes the login prompt is ripped apart by asynchronous daemon updates
		res, err := expecter.ExpectBatch(b, 1*time.Minute)
		if err != nil {
			log.DefaultLogger().Object(vmi).Reason(err).Errorf("Retried login attempt after two minutes failed: %+v", res)
			return err
		}
	}

	err = configureConsole(expecter, false)
	if err != nil {
		return err
	}
	return nil
}

// Rest below is copied from kubevirt.io/kubevirt/tests/console

func configureConsole(expecter expect.Expecter, shouldSudo bool) error {
	sudoString := ""
	if shouldSudo {
		sudoString = "sudo "
	}
	batch := append([]expect.Batcher{
		&expect.BSnd{S: "stty cols 500 rows 500\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
		&expect.BSnd{S: fmt.Sprintf("%sdmesg -n 1\n", sudoString)},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")}})
	resp, err := expecter.ExpectBatch(batch, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Infof("%v", resp)
	}
	return err
}
