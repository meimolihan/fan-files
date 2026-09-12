const name: string = window.fanFiles.Name || "fan-files";
const disableExternal: boolean = window.fanFiles.DisableExternal;
const disableUsedPercentage: boolean = window.fanFiles.DisableUsedPercentage;
const baseURL: string = window.fanFiles.BaseURL;
const staticURL: string = window.fanFiles.StaticURL;
const recaptcha: string = window.fanFiles.ReCaptcha;
const recaptchaKey: string = window.fanFiles.ReCaptchaKey;
const signup: boolean = window.fanFiles.Signup;
const version: string = window.fanFiles.Version;
const logoURL = `${staticURL}/img/logo.svg`;
const noAuth: boolean = window.fanFiles.NoAuth;
const authMethod = window.fanFiles.AuthMethod;
const logoutPage: string = window.fanFiles.LogoutPage;
const loginPage: boolean = window.fanFiles.LoginPage;
const theme: UserTheme = window.fanFiles.Theme;
const enableThumbs: boolean = window.fanFiles.EnableThumbs;
const resizePreview: boolean = window.fanFiles.ResizePreview;
const enableExec: boolean = window.fanFiles.EnableExec;
const tusSettings = window.fanFiles.TusSettings;
const origin = window.location.origin;
const tusEndpoint = `/api/tus`;
const hideLoginButton = window.fanFiles.HideLoginButton;

export {
  name,
  disableExternal,
  disableUsedPercentage,
  baseURL,
  logoURL,
  recaptcha,
  recaptchaKey,
  signup,
  version,
  noAuth,
  authMethod,
  logoutPage,
  loginPage,
  theme,
  enableThumbs,
  resizePreview,
  enableExec,
  tusSettings,
  origin,
  tusEndpoint,
  hideLoginButton,
};
