package browser

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/term"
)

type Website string

const (
	AWSConsole  Website = ""
	AzurePortal Website = "https://portal.azure.com"
)

// The AWS console will prevent automatically by push a feedback pop up based on their security design
// So the function only fills username and password, then user can click login button by themselves.
func loginAWSConsole(url, username, password string) chromedp.Tasks {
	usernameInputSel := `input[name="username"], input[type="username"]`
	passwordInputSel := `input[name="password"], input[type="password"]`

	return chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitVisible(usernameInputSel, chromedp.ByQuery),
		chromedp.SendKeys(usernameInputSel, username, chromedp.ByQuery),
		chromedp.WaitVisible(passwordInputSel, chromedp.ByQuery),
		chromedp.SendKeys(passwordInputSel, password, chromedp.ByQuery),
	}
}

// The Azure portal has a more straightforward login flow, so we can automate the entire process.
func loginAzurePortal(username, password string) chromedp.Tasks {
	usernameInputSel := `input[name="loginfmt"], input[type="email"]`
	passwordInputSel := `input[name="accesspass"], #accesspass`
	loginBtnSel := `document.querySelector('input[type=submit]')`

	return chromedp.Tasks{
		chromedp.Navigate(string(AzurePortal)),
		chromedp.WaitVisible(usernameInputSel, chromedp.ByQuery),
		chromedp.SendKeys(usernameInputSel, username, chromedp.ByQuery),
		clickIfExistsJS(loginBtnSel),
		chromedp.WaitVisible(passwordInputSel, chromedp.ByQuery),
		chromedp.SendKeys(passwordInputSel, password, chromedp.ByQuery),
		clickIfExistsJS(loginBtnSel),
		chromedp.Sleep(3 * time.Second),
	}
}

func clickIfExistsJS(query string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		var exists bool

		// Using JavaScript to check if query selector returns an element
		checkJS := fmt.Sprintf(`%s !== null`, query)
		if err := chromedp.Evaluate(checkJS, &exists).Do(ctx); err != nil {
			return err
		}

		// If there's nothing to click
		if !exists {
			return nil
		}

		// Make sure the element is interactable before clicking
		clickJS := fmt.Sprintf(`%s && %s.click()`, query, query)
		return chromedp.Evaluate(clickJS, nil).Do(ctx)
	}
}

func LoginInBrowser(username, password string, website Website, url string) error {
	session, err := startLoginInBrowser(username, password, website, url)
	if err != nil {
		return err
	}
	defer session.close()

	fmt.Println("Browser is open. You may continue interacting manually.")
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	if interactive {
		fmt.Println("Press ENTER or close the browser to finish.")
	}
	waitForBrowser(session.done, interactive, os.Stdin)
	return nil
}

// StartLoginInBrowser opens an interactive browser session and returns a
// function that closes it. Callers should retain the returned function for as
// long as the browser must remain open.
func StartLoginInBrowser(username, password string, website Website, url string) (func(), error) {
	session, err := startLoginInBrowser(username, password, website, url)
	if err != nil {
		return nil, err
	}
	return session.close, nil
}

type loginBrowserSession struct {
	close func()
	done  <-chan struct{}
}

func startLoginInBrowser(username, password string, website Website, url string) (loginBrowserSession, error) {
	allocCtx, cancelAllocator := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
			chromedp.Flag("incognito", true))...)

	ctx, cancelBrowser := chromedp.NewContext(allocCtx)
	closeBrowser := func() {
		cancelBrowser()
		cancelAllocator()
	}

	switch website {
	case AWSConsole:
		if err := chromedp.Run(ctx, loginAWSConsole(url, username, password)); err != nil {
			closeBrowser()
			return loginBrowserSession{}, err
		}
	case AzurePortal:
		if err := chromedp.Run(ctx, loginAzurePortal(username, password)); err != nil {
			closeBrowser()
			return loginBrowserSession{}, err
		}
	default:
		closeBrowser()
		return loginBrowserSession{}, fmt.Errorf("unsupported login website %q", website)
	}

	chromedpContext := chromedp.FromContext(ctx)
	if chromedpContext == nil || chromedpContext.Browser == nil {
		closeBrowser()
		return loginBrowserSession{}, fmt.Errorf("browser session did not start")
	}

	return loginBrowserSession{
		close: closeBrowser,
		done:  chromedpContext.Browser.LostConnection,
	}, nil
}

func waitForBrowser(done <-chan struct{}, interactive bool, input io.Reader) {
	if !interactive {
		<-done
		return
	}

	enter := make(chan struct{})
	go func() {
		_, _ = fmt.Fscanln(input)
		close(enter)
	}()

	select {
	case <-done:
	case <-enter:
	}
}
