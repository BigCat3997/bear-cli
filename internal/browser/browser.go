package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
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

func LoginInBrowser(username, password string, website Website, url string) {
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
			chromedp.Flag("incognito", true),
		)...)

	ctx, _ := chromedp.NewContext(allocCtx)

	switch website {
	case AWSConsole:
		if err := chromedp.Run(ctx, loginAWSConsole(url, username, password)); err != nil {
			fmt.Println("Error:", err)
		}
	case AzurePortal:
		if err := chromedp.Run(ctx, loginAzurePortal(username, password)); err != nil {
			fmt.Println("Error:", err)
		}
	}

	fmt.Println("Browser is open. You may continue interacting manually.")
	fmt.Println("Press ENTER to terminate Go process (browser will close).")

	// Keep program alive until user decides
	fmt.Scanln()
}
