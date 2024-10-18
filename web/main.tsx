import { Colors, OverlaysProvider } from '@blueprintjs/core'
import '@blueprintjs/core/lib/css/blueprint.css'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './styles.css'

const style = document.querySelector<HTMLHtmlElement>(':root')!.style
style.setProperty('--white', Colors.WHITE)
style.setProperty('--blue3', Colors.BLUE3)
style.setProperty('--gray3', Colors.GRAY3)
style.setProperty('--light-gray4', Colors.LIGHT_GRAY4)

ReactDOM.createRoot(document.getElementById('root')!).render(
  <OverlaysProvider>
    <App />
  </OverlaysProvider>,
)
