// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.648
package pages

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import "github.com/heyztb/lists-backend/internal/html/templates/shared"
import "github.com/heyztb/lists-backend/internal/html/templates/partials"

func Login() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
		if !templ_7745c5c3_IsBuffer {
			templ_7745c5c3_Buffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		templ_7745c5c3_Var2 := templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
			templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
			if !templ_7745c5c3_IsBuffer {
				templ_7745c5c3_Buffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<!--\n    // v0 by Vercel.\n    // https://v0.dev/t/Ui6lPpyAQKV\n    --> <script type=\"module\" src=\"/assets/srp.js\">\n\t\t  const onSubmit = async (event) => {\n\t\t    event.preventDefault()\n\t\t    const form = new FormData(event.target)\n\t\t    const data = Object.fromEntries(form)\n\t\t    console.log(data)\n\t\t  }\n\t\t</script> <div class=\"w-full lg:grid lg:min-h-[600px] lg:grid-cols-2 xl:min-h-[800px]\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			templ_7745c5c3_Err = partials.NavAuth(false).Render(ctx, templ_7745c5c3_Buffer)
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"hidden bg-muted lg:block\"><img src=\"/placeholder.svg\" alt=\"Image\" width=\"1920\" height=\"1080\" class=\"h-full w-full object-cover dark:brightness-[0.2] dark:grayscale\" style=\"aspect-ratio: 1920 / 1080; object-fit: cover;\"></div><div class=\"flex items-center justify-center py-12\"><div class=\"mx-auto grid w-[350px] gap-6\"><div class=\"grid gap-2 text-center\"><h1 class=\"text-3xl font-bold\">Login</h1><p class=\"text-balance text-muted-foreground\">Enter your email below to login to your account</p></div><form class=\"grid gap-4\" onsubmit=\"onSubmit(this)\" method=\"POST\"><div class=\"grid gap-2\"><label class=\"text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70\" for=\"email\">Email</label> <input class=\"flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50\" name=\"email\" id=\"email\" placeholder=\"m@example.com\" required=\"\" type=\"email\"></div><div class=\"grid gap-2\"><div class=\"flex items-center\"><label class=\"text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70\" for=\"password\">Password</label> <a class=\"ml-auto inline-block text-sm underline\" href=\"#\">Forgot your password?</a></div><input class=\"flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50\" name=\"password\" id=\"password\" required=\"\" type=\"password\"></div><div class=\"flex flex-col gap-2\"><button class=\"inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2 flex-1\" type=\"submit\">Login</button></div></form></div></div></div>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			if !templ_7745c5c3_IsBuffer {
				_, templ_7745c5c3_Err = io.Copy(templ_7745c5c3_W, templ_7745c5c3_Buffer)
			}
			return templ_7745c5c3_Err
		})
		templ_7745c5c3_Err = shared.Page("Login").Render(templ.WithChildren(ctx, templ_7745c5c3_Var2), templ_7745c5c3_Buffer)
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}
