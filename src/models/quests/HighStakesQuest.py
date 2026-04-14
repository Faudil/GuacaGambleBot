from src.models.Quest import Quest, QuestStep, QuestStepType, QuestType, QuestReward, QuestRegistry
from typing import List, Dict, Any

@QuestRegistry.register
class HighStakesQuest(Quest):
    id = "high_stakes"
    type = QuestType.SIDE
    npc_id = "gamblebot"
    reputation_req = 2 # Requires GambleBot level 2
    
    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                QuestStepType.DIALOGUE,
                "quests.high_stakes.step0",
                rewards=None
            ),
            QuestStep(
                QuestStepType.ACTIVITY,
                "quests.high_stakes.step1",
                target_count=5000,
                target_stat="casino_won",
                rewards=QuestReward(money=2000, xp=500, crowns=5)
            ),
            QuestStep(
                QuestStepType.DIALOGUE,
                "quests.high_stakes.step2",
                rewards=QuestReward(items=["jeton de casino", "jeton de casino"])
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx == 1 and activity == "casino_won":
            current_count = current_progress.get('progress_value', 0)
            target = self.steps[step_idx].extra.get('target_count', 5000)
            
            # For money-based stats, we check the total amount won
            if current_count + amount >= target:
                return True
        return False
