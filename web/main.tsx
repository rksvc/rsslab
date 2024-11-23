import { Colors, OverlaysProvider } from '@blueprintjs/core'
import '@blueprintjs/core/lib/css/blueprint.css'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './styles.css'

const style = document.querySelector<HTMLHtmlElement>(':root')!.style
style.setProperty('--black', Colors.BLACK)
style.setProperty('--gray1', Colors.GRAY1)
style.setProperty('--dark-gray3', Colors.DARK_GRAY3)

ReactDOM.createRoot(document.getElementById('root')!).render(
  <OverlaysProvider>
    <App />
  </OverlaysProvider>,
)
