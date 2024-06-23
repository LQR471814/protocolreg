package protocolreg

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/ini.v1"
)

type LinuxOpenQuickPreview struct {
	Name      string
	Exec      string
	TryExec   string
	NoDisplay bool
}

func (opts LinuxOpenQuickPreview) WriteTo(f *ini.File) error {
	section, err := f.NewSection("Desktop Action QuickPreview")
	if err != nil {
		return err
	}

	_, err = section.NewKey("Name", opts.Name)
	if err != nil {
		return err
	}
	_, err = section.NewKey("Exec", opts.Exec)
	if err != nil {
		return err
	}
	_, err = section.NewKey("TryExec", opts.TryExec)
	if err != nil {
		return err
	}
	_, err = section.NewKey("NoDisplay", fmt.Sprint(opts.NoDisplay))
	if err != nil {
		return err
	}

	return nil
}

type LinuxOpenWithTerminal struct {
	Name string
	Exec string
}

func (opts LinuxOpenWithTerminal) WriteTo(f *ini.File) error {
	section, err := f.NewSection("Desktop Action OpenWithTerminal")
	if err != nil {
		return err
	}

	_, err = section.NewKey("Name", opts.Name)
	if err != nil {
		return err
	}
	_, err = section.NewKey("Exec", opts.Exec)
	if err != nil {
		return err
	}

	return nil
}

type LinuxMetadataOptions struct {
	Name       string
	Comment    string
	Icon       string
	Categories []string
}

type LinuxOptions struct {
	Metadata LinuxMetadataOptions

	Exec      string
	Protocols []string
	Mimetypes []string

	OpenWithTerminal  LinuxOpenWithTerminal
	OpenQuickPreview  LinuxOpenQuickPreview
	NoUrlArgNecessary bool
}

func (opts LinuxOptions) AllMimetypes() []string {
	mimetypes := make([]string, len(opts.Protocols)+len(opts.Mimetypes))
	for i := 0; i < len(opts.Protocols); i++ {
		mimetypes[i] = fmt.Sprintf("x-scheme-handler/%s", opts.Protocols[i])
	}
	for i := 0; i < len(opts.Mimetypes); i++ {
		mimetypes[i+len(opts.Mimetypes)] = opts.Mimetypes[i]
	}
	return mimetypes
}

func (opts LinuxOptions) WriteTo(f *ini.File) error {
	section, err := f.NewSection("Desktop Entry")
	if err != nil {
		return err
	}

	_, err = section.NewKey("Type", "Application")
	if err != nil {
		return err
	}
	_, err = section.NewKey("StartupNotify", "false")
	if err != nil {
		return err
	}
	_, err = section.NewKey("Name", opts.Metadata.Name)
	if err != nil {
		return err
	}
	_, err = section.NewKey("Comment", opts.Metadata.Comment)
	if err != nil {
		return err
	}
	_, err = section.NewKey("Icon", opts.Metadata.Icon)
	if err != nil {
		return err
	}
	_, err = section.NewKey("Exec", opts.Exec)
	if err != nil {
		return err
	}
	_, err = section.NewKey("Categories", strings.Join(opts.Metadata.Categories, ";"))
	if err != nil {
		return err
	}

	_, err = section.NewKey("MimeType", strings.Join(opts.AllMimetypes(), ";"))
	if err != nil {
		return err
	}

	opts.OpenQuickPreview.WriteTo(f)
	opts.OpenWithTerminal.WriteTo(f)

	return nil
}

func linuxIdToDesktopFilename(id string) string {
	return fmt.Sprintf("%s-opener.desktop", id)
}

func RegisterLinux(id string, opts LinuxOptions) error {
	if id == "" {
		return fmt.Errorf("you must specify a non-empty id that is a unique identifier for this registration")
	}
	if len(opts.Protocols) == 0 {
		return fmt.Errorf("you must specify at least one protocol")
	}

	if opts.Exec == "" {
		return fmt.Errorf("you must specify a command for Exec, it will be executed when your application is called via custom url")
	}
	if !opts.NoUrlArgNecessary && !strings.Contains(opts.Exec, "%u") && !strings.Contains(opts.Exec, "%U") {
		return fmt.Errorf("you have not specified %%u or %%U in your Exec command, %%u/U is the placeholder for the specific url your application is called with. if you meant to omit this set the NoUrlArgNecessary option to true")
	}

	homedir, ok := os.LookupEnv("HOME")
	if !ok {
		return fmt.Errorf("environment variable not set: $HOME")
	}
	appsdir := path.Join(homedir, ".local", "share", "applications")
	err := os.MkdirAll(appsdir, 0666)
	if err != nil {
		return err
	}
	filename := linuxIdToDesktopFilename(id)

	file, err := os.Create(path.Join(appsdir, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	desktopFile := ini.Empty(ini.LoadOptions{
		// to prevent all lines with semicolons in it from getting surrounded with backticks
		// which will make update-desktop-database fail to parse it
		IgnoreInlineComment: true,
	})
	err = opts.WriteTo(desktopFile)
	if err != nil {
		return err
	}
	_, err = desktopFile.WriteTo(file)
	if err != nil {
		return err
	}

	cmd := exec.Command("xdg-mime", append(
		[]string{"default", filename},
		opts.AllMimetypes()...,
	)...)
	err = cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command(
		"update-desktop-database",
		path.Join(homedir, ".local", "share", "applications"),
	)
	err = cmd.Run()

	return err
}

func UnregisterLinux(id string) error {
	homedir, ok := os.LookupEnv("HOME")
	if !ok {
		return fmt.Errorf("environment variable not set: $HOME")
	}
	appsdir := path.Join(homedir, ".local", "share", "applications")
	err := os.MkdirAll(appsdir, 0666)
	if err != nil {
		return err
	}
	filename := linuxIdToDesktopFilename(id)

	err = os.Remove(path.Join(appsdir, filename))
	if err != nil {
		return err
	}

	defaultAppsFilepath := path.Join(homedir, ".config", "mimeapps.list")
	defaultAppsList, err := ini.Load(defaultAppsFilepath)
	section, err := defaultAppsList.GetSection("Default Applications")
	if ini.IsErrDelimiterNotFound(err) {
		return nil
	}

	toBeDeleted := []string{}
	for _, kv := range section.Keys() {
		if kv.String() == filename {
			toBeDeleted = append(toBeDeleted, kv.Name())
		}
	}
	for _, key := range toBeDeleted {
		section.DeleteKey(key)
	}

	f, err := os.Create(defaultAppsFilepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = defaultAppsList.WriteTo(f)
	if err != nil {
		return err
	}

	return nil
}
