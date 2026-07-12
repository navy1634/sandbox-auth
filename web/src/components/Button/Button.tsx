export type ButtonProps = {
  isActive?: boolean;
  primary?: boolean;
  label?: string;
  backgroundColor?: string;
};

const Button: React.FC<ButtonProps> = ({ isActive, label }) => (
  <button
    className={`button ${isActive ? "active" : ""}`}
    onClick={() => console.log("Button clicked!")}
  >
    {label}
  </button>
);

export default Button;
