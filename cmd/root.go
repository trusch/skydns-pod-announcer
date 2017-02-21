// Copyright © 2017 Tino Rusch
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "skydns-pod-announcer",
	Short: "anounce your pod ip to skydns",
	Long:  `This searches for the first non-local ip and announces it to skydns.`,
	Run: func(cmd *cobra.Command, args []string) {
		hostname := viper.GetString("hostname")
		etcd := viper.GetString("etcd")
		ip := viper.GetString("ip")
		if hostname == "" {
			h, err := os.Hostname()
			if err != nil {
				log.Fatal(err)
			}
			hostname = h
			log.Printf("no hostname given, using %v", hostname)
		}
		if ip == "" {
			i, err := getIP()
			if err != nil {
				log.Fatal(err)
			}
			ip = i
			log.Printf("no ip given, using %v", ip)
		}
		err := announceIP(ip, hostname, etcd)
		if err != nil {
			log.Fatal(err)
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.skydns-pod-announcer.yaml)")
	RootCmd.Flags().String("hostname", "", "hostname to announce")
	RootCmd.Flags().String("etcd", "http://etcd:2379", "etcd endpoint")
	RootCmd.Flags().String("ip", "", "ip to announce")

	viper.BindPFlags(RootCmd.Flags())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	viper.SetConfigName(".skydns-pod-announcer") // name of config file (without extension)
	viper.AddConfigPath("$HOME")                 // adding home directory as first search path
	viper.AutomaticEnv()                         // read in environment variables that match

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func getIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		a := addr.String()
		if a != "127.0.0.1/8" {
			return strings.Split(a, "/")[0], nil
		}
	}
	return "", errors.New("not found")
}

func announceIP(ip, name, etcdAddr string) error {
	body := strings.NewReader(fmt.Sprintf(`{"host":"%v"}`, ip))
	req, err := http.NewRequest("PUT", fmt.Sprintf("%v/skydns/local/skydns/%v", etcdAddr, name), body)
	if err != nil {
		return err
	}
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("fail")
	}
	return nil
}
