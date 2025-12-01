import { Colors, OverlaysProvider } from '@blueprintjs/core'
import '@blueprintjs/core/lib/css/blueprint.css'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './styles.css'

const style = document.querySelector<HTMLHtmlElement>(':root')!.style
style.setProperty('--black', Colors.BLACK)
style.setProperty('--gray1', Colors.GRAY1)
style.setProperty('--gray4', Colors.GRAY4)
style.setProperty('--light-gray5', Colors.LIGHT_GRAY5)
style.setProperty('--dark-gray2', Colors.DARK_GRAY2)

ReactDOM.createRoot(document.getElementById('root')!).render(
  <OverlaysProvider>
    <App />
  </OverlaysProvider>,
)
